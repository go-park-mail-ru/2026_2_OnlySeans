package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
	"github.com/google/uuid"
)

type UserUseCase interface {
	Create(ctx context.Context, user domain.User, password string) (int64, error)
	GetByID(ctx context.Context, userID int64) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateProfile(ctx context.Context, userID int64, name, email *string) (domain.User, error)
	GetAvatarURL(ctx context.Context, userID int64) (string, error)
	GetAvatarURLByFilename(ctx context.Context, filename string) (string, error)
	UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type userUseCase struct {
	userRepo       repository.UserRepository
	avatarRepo     repository.AvatarRepository
	fileStorage    FileStorage
	passwordHasher PasswordHasher
	appLogger      logger.Logger
}

func NewUserUseCase(
	users repository.UserRepository,
	avatars repository.AvatarRepository,
	files FileStorage,
	hasher PasswordHasher,
	appLogger logger.Logger,
) UserUseCase {
	return &userUseCase{
		userRepo:       users,
		avatarRepo:     avatars,
		fileStorage:    files,
		passwordHasher: hasher,
		appLogger:      appLogger,
	}
}

func (u *userUseCase) Create(ctx context.Context, user domain.User, password string) (int64, error) {
	user.Email = strings.TrimSpace(user.Email)
	user.Username = strings.TrimSpace(user.Username)

	if err := utils.ValidatePassword(password); err != nil {
		return 0, registrationValidationError(domain.ErrInvalidPassword, err)
	}

	if err := utils.ValidateEmail(user.Email); err != nil {
		return 0, registrationValidationError(domain.ErrInvalidEmail, err)
	}

	if err := utils.ValidateUsername(user.Username); err != nil {
		return 0, registrationValidationError(domain.ErrInvalidUsername, err)
	}

	hashedPassword, err := u.passwordHasher.Hash(password)
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

func registrationValidationError(fieldErr, err error) error {
	reason := strings.TrimPrefix(err.Error(), fieldErr.Error()+": ")
	return fmt.Errorf("%w: %s", fieldErr, reason)
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

	return u.getAvatarURL(ctx, avatar.Filename)
}

var (
	avatarFilenamePattern         = regexp.MustCompile(`^[0-9a-f]{32}\.(jpg|png)$`)
	prefixedAvatarFilenamePattern = regexp.MustCompile(`^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}\.(jpeg|png|gif)$`)
)

func (u *userUseCase) GetAvatarURLByFilename(ctx context.Context, filename string) (string, error) {
	switch {
	case avatarFilenamePattern.MatchString(filename):
		return u.getAvatarURL(ctx, filename)
	case prefixedAvatarFilenamePattern.MatchString(filename):
		return u.getAvatarURL(ctx, "avatars/"+filename)
	default:
		return "", domain.ErrInvalidAvatarName
	}
}

func (u *userUseCase) getAvatarURL(ctx context.Context, key string) (string, error) {
	url, err := u.fileStorage.GetURL(ctx, key)
	if errors.Is(err, domain.ErrFileNotFound) {
		return "", domain.ErrAvatarNotFound
	}

	return url, err
}

func (u *userUseCase) UpdateAvatar(ctx context.Context, userID int64, imageData []byte) (string, error) {
	if len(imageData) > imageutil.MaxAvatarSize {
		return "", domain.ErrAvatarTooLarge
	}

	user, err := u.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	format, err := imageutil.ValidateAvatar(imageData)
	if err != nil {
		return "", domain.ErrInvalidImage
	}

	contentType := "image/" + format
	extension := ".png"
	if format == "jpeg" {
		extension = ".jpg"
	}
	filename := strings.ReplaceAll(uuid.NewString(), "-", "") + extension

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
