package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	passUtil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	"github.com/google/uuid"
)

type UserUseCase interface {
	Create(ctx context.Context, user domain.User, password string) (int64, error)
	GetByID(ctx context.Context, userID int64) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, name, email *string) (domain.User, error)
	GetAvatarURL(ctx context.Context, userID int64) (string, error)
	UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type userUseCase struct {
	userRepo    repository.UserRepository
	avatarRepo  repository.AvatarRepository
	fileStorage s3.FileStorage
	appLogger   logger.Logger
}

func NewUserUseCase(ur repository.UserRepository, avatars repository.AvatarRepository, fs s3.FileStorage, l logger.Logger) UserUseCase {
	return &userUseCase{
		userRepo:    ur,
		avatarRepo:  avatars,
		fileStorage: fs,
		appLogger:   l,
	}
}

func (u *userUseCase) Create(ctx context.Context, user domain.User, password string) (int64, error) {
	user.Email = strings.TrimSpace(user.Email)
	user.Username = strings.TrimSpace(user.Username)

	if err := utils.ValidateEmail(user.Email); err != nil {
		return 0, domain.ErrInvalidEmail
	}

	if err := utils.ValidateUsername(user.Username); err != nil {
		return 0, domain.ErrInvalidUsername
	}

	hashedPassword, err := (&passUtil.BcryptPasswordService{}).Hash(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword

	id, err := u.userRepo.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return 0, err
		}

		return 0, fmt.Errorf("failed to create user in db: %w", err)
	}

	return id, nil
}

func (u *userUseCase) GetByID(ctx context.Context, userID int64) (domain.User, error) {
	user, err := u.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, err
		}

		return domain.User{}, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

func (u *userUseCase) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := u.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, err
		}

		return domain.User{}, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (u *userUseCase) UpdateProfile(ctx context.Context, userID int64, name, email *string) (domain.User, error) {
	if email != nil {
		value := strings.TrimSpace(*email)

		if err := utils.ValidateEmail(value); err != nil {
			return domain.User{}, domain.ErrInvalidEmail
		}

		email = &value
	}

	if name != nil {
		value := strings.TrimSpace(*name)

		if err := utils.ValidateUsername(value); err != nil {
			return domain.User{}, domain.ErrInvalidUsername
		}

		name = &value
	}

	user, err := u.userRepo.UpdateProfile(ctx, userID, name, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrEmailAlreadyExists) {
			return domain.User{}, err
		}
		return domain.User{}, fmt.Errorf("failed to update user profile: %w", err)
	}

	return user, nil
}

func (u *userUseCase) GetAvatarURL(ctx context.Context, userID int64) (string, error) {
	avatar, err := u.avatarRepo.GetAvatar(ctx, userID)
	if err != nil {
		return "", err
	}

	return u.fileStorage.GetURL(ctx, avatar.Filename)
}

func (u *userUseCase) UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error) {
	user, err := u.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	format, err := imageutil.Validate(imageData)
	if err != nil {
		return "", domain.ErrInvalidImage
	}

	contentType := "image/" + format
	filename := "avatars/" + uuid.NewString() + "." + format

	if err := u.fileStorage.UploadFile(ctx, filename, contentType, imageData); err != nil {
		return "", fmt.Errorf("upload avatar: %w", err)
	}

	file := domain.AvatarFile{
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(imageData)),
	}
	if err := u.avatarRepo.SaveAvatar(ctx, userID, file); err != nil {
		saveErr := fmt.Errorf("save avatar metadata: %w", err)
		if errors.Is(err, domain.ErrCommitUnknown) {
			return "", saveErr
		}
		return "", errors.Join(saveErr, u.deleteAvatarFile(ctx, filename))
	}

	if user.AvatarFilename != nil && *user.AvatarFilename != "" {
		u.cleanupOldAvatar(ctx, *user.AvatarFilename)
	}

	return filename, nil
}

func (u *userUseCase) DeleteAvatar(ctx context.Context, userID int64) error {
	user, err := u.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.AvatarFilename == nil || *user.AvatarFilename == "" {
		return nil
	}

	if err := u.avatarRepo.DeleteAvatar(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return err
		}
		return fmt.Errorf("failed to reset avatar in db: %w", err)
	}

	u.cleanupOldAvatar(ctx, *user.AvatarFilename)

	return nil
}

func (u *userUseCase) cleanupOldAvatar(ctx context.Context, filename string) {
	if err := u.deleteAvatarFile(ctx, filename); err != nil && u.appLogger != nil {
		u.appLogger.WarnContext(ctx, "could not remove old avatar", map[string]any{
			"filename": filename,
			"error":    err,
		})
	}
}

func (u *userUseCase) deleteAvatarFile(ctx context.Context, filename string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := u.fileStorage.DeleteFile(cleanupCtx, filename); err != nil {
		return fmt.Errorf("delete avatar object: %w", err)
	}
	return nil
}
