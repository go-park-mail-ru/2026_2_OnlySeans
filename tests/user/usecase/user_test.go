package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"testing"

	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
)

type userRepoStub struct {
	repository.UserRepository
	create func(context.Context, domain.User) (int64, error)
	get    func(context.Context, int64) (domain.User, error)
}

func (s userRepoStub) CreateUser(ctx context.Context, user domain.User) (int64, error) {
	return s.create(ctx, user)
}
func (s userRepoStub) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return s.get(ctx, id)
}

type avatarRepoStub struct {
	repository.AvatarRepository
	save func(context.Context, int64, domain.AvatarFile) error
}

func (s avatarRepoStub) SaveAvatar(ctx context.Context, id int64, file domain.AvatarFile) error {
	return s.save(ctx, id, file)
}

type fileStorageStub struct {
	usecase.FileStorage
	upload func(context.Context, string, string, []byte) error
	delete func(context.Context, string) error
}

func (s fileStorageStub) UploadFile(ctx context.Context, key, mime string, data []byte) error {
	return s.upload(ctx, key, mime, data)
}
func (s fileStorageStub) DeleteFile(ctx context.Context, key string) error { return s.delete(ctx, key) }

func TestCreateUser(t *testing.T) {
	failure := errors.New("database failure")
	hashFailure := errors.New("hashing failed")

	tests := []struct {
		name       string
		email      string
		username   string
		password   string
		hashErr    error
		wantHashes int
		repoErr    error
		wantErr    error
		wantCalls  int
	}{
		{
			name:       "valid registration preserves case and trims whitespace",
			email:      " Mixed@example.test ",
			username:   " Anna ",
			password:   "Password1",
			wantCalls:  1,
			wantHashes: 1,
		},
		{
			name:     "invalid email",
			email:    "invalid",
			username: "Anna",
			password: "Password1",
			wantErr:  domain.ErrInvalidEmail,
		},
		{
			name:     "invalid username",
			email:    "reader@example.test",
			username: "",
			password: "Password1",
			wantErr:  domain.ErrInvalidUsername,
		},
		{
			name:     "invalid password",
			email:    "reader@example.test",
			username: "Anna",
			password: "short",
			wantErr:  domain.ErrInvalidPassword,
		},
		{
			name:       "hashing failure",
			email:      "reader@example.test",
			username:   "Anna",
			password:   "Password1",
			hashErr:    hashFailure,
			wantErr:    hashFailure,
			wantHashes: 1,
		},
		{
			name:       "email conflict",
			email:      "reader@example.test",
			username:   "Anna",
			password:   "Password1",
			repoErr:    domain.ErrUserExists,
			wantErr:    domain.ErrUserExists,
			wantCalls:  1,
			wantHashes: 1,
		},
		{
			name:       "username conflict",
			email:      "reader@example.test",
			username:   "Anna",
			password:   "Password1",
			repoErr:    domain.ErrUsernameTaken,
			wantErr:    domain.ErrUsernameTaken,
			wantCalls:  1,
			wantHashes: 1,
		},
		{
			name:       "database failure",
			email:      "reader@example.test",
			username:   "Anna",
			password:   "Password1",
			repoErr:    failure,
			wantErr:    failure,
			wantCalls:  1,
			wantHashes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := mocks.NewMockPasswordHasher(gomock.NewController(t))
			hasher.EXPECT().Hash(tt.password).Return("stored hash", tt.hashErr).Times(tt.wantHashes)

			calls := 0
			repo := userRepoStub{
				create: func(_ context.Context, user domain.User) (int64, error) {
					calls++
					require.Equal(t, "stored hash", user.PasswordHash)
					if tt.wantErr == nil {
						require.Equal(t, "Mixed@example.test", user.Email)
						require.Equal(t, "Anna", user.Username)
					}
					return 7, tt.repoErr
				},
			}
			uc := usecase.NewUserUseCase(repo, nil, nil, hasher, nil)
			id, err := uc.Create(t.Context(), domain.User{
				Email:    tt.email,
				Username: tt.username,
			}, tt.password)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantCalls, calls)

			if tt.wantErr == nil {
				require.EqualValues(t, 7, id)
			} else {
				require.Zero(t, id)
			}
		})
	}
}

func TestAvatarUploadLifecycle(t *testing.T) {
	var pngData bytes.Buffer
	require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2))))

	failure := errors.New("dependency failure")

	tests := []struct {
		name       string
		data       []byte
		getErr     error
		uploadErr  error
		saveErr    error
		cleanupErr error
		wantErr    error
		calls      []string
	}{
		{
			name:  "upload and attach",
			data:  pngData.Bytes(),
			calls: []string{"user", "upload", "save"},
		},
		{
			name:    "missing account",
			data:    pngData.Bytes(),
			getErr:  domain.ErrUserNotFound,
			wantErr: domain.ErrUserNotFound,
			calls:   []string{"user"},
		},
		{
			name:    "invalid image",
			data:    []byte("invalid"),
			wantErr: domain.ErrInvalidImage,
			calls:   []string{"user"},
		},
		{
			name:    "oversized image",
			data:    make([]byte, imageutil.MaxAvatarSize+1),
			wantErr: domain.ErrAvatarTooLarge,
		},
		{
			name:      "storage fails before metadata",
			data:      pngData.Bytes(),
			uploadErr: failure,
			wantErr:   failure,
			calls:     []string{"user", "upload"},
		},
		{
			name:    "metadata failure removes uploaded object",
			data:    pngData.Bytes(),
			saveErr: failure,
			wantErr: failure,
			calls:   []string{"user", "upload", "save", "delete"},
		},
		{
			name:    "ambiguous commit preserves object",
			data:    pngData.Bytes(),
			saveErr: domain.ErrCommitUnknown,
			wantErr: domain.ErrCommitUnknown,
			calls:   []string{"user", "upload", "save"},
		},
		{
			name:       "cleanup failure preserves original error",
			data:       pngData.Bytes(),
			saveErr:    domain.ErrUserNotFound,
			cleanupErr: failure,
			wantErr:    domain.ErrUserNotFound,
			calls:      []string{"user", "upload", "save", "delete"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			var uploadedKey string
			users := userRepoStub{
				get: func(_ context.Context, id int64) (domain.User, error) {
					calls = append(calls, "user")
					require.EqualValues(t, 7, id)
					return domain.User{}, tt.getErr
				},
			}
			storage := fileStorageStub{
				upload: func(_ context.Context, key, mime string, data []byte) error {
					calls = append(calls, "upload")
					uploadedKey = key
					require.Equal(t, "image/png", mime)
					require.Equal(t, tt.data, data)
					return tt.uploadErr
				},
				delete: func(_ context.Context, key string) error {
					calls = append(calls, "delete")
					require.Equal(t, uploadedKey, key)
					return tt.cleanupErr
				},
			}
			avatars := avatarRepoStub{
				save: func(_ context.Context, id int64, file domain.AvatarFile) error {
					calls = append(calls, "save")
					require.EqualValues(t, 7, id)
					require.Equal(t, uploadedKey, file.Filename)
					require.Equal(t, "image/png", file.ContentType)
					require.EqualValues(t, len(tt.data), file.Size)
					return tt.saveErr
				},
			}
			uc := usecase.NewUserUseCase(users, avatars, storage, mocks.NewMockPasswordHasher(gomock.NewController(t)), nil)
			filename, err := uc.UpdateAvatar(t.Context(), 7, tt.data)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.calls, calls)

			if tt.wantErr == nil {
				require.Regexp(t, `^[0-9a-f]{32}\.png$`, filename)
			} else {
				require.Empty(t, filename)
			}
		})
	}
}
