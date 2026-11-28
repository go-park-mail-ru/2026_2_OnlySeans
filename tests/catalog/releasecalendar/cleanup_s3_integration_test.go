//go:build integration

package releasecalendar_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	postgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/postgres"
	posters3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	userfixture "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/integrationfixture"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/filecleanup"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func TestConcurrentAvatarsAndPosterCleanupInS3(t *testing.T) {
	db := cleanupDB(t)
	storage, _ := testutil.IntegrationS3(t)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	users := pb.NewUserServiceClient(userfixture.New(t, db, storage))
	created, err := users.CreateUser(ctx, &pb.CreateUserRequest{
		Email:    "cleanup@example.test",
		Username: "cleanup",
		Password: "Password123",
	})
	require.NoError(t, err)

	user := created.GetUserId()
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))

	keys := make(chan string, 8)

	t.Cleanup(func() {
		close(keys)

		cleanCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()

		for key := range keys {
			require.NoError(t, storage.DeleteFile(cleanCtx, key))
		}
	})

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			response, err := users.UpdateAvatar(ctx, &pb.UpdateAvatarRequest{
				UserId:    user,
				ImageData: imageData.Bytes(),
			})
			if err != nil {
				t.Errorf("concurrent avatar upload: %v", err)
				return
			}
			keys <- response.GetAvatarFilename()
		})
	}

	wg.Wait()
	require.False(t, t.Failed())

	worker := filecleanup.New(db, storage)
	drain := func() {
		for range 20 {
			done, err := worker.ProcessOne(ctx)
			require.NoError(t, err)

			if !done {
				return
			}
		}

		t.Fatal("cleanup queue did not drain")
	}
	drain()

	profile, err := users.GetByID(ctx, &pb.GetByIDRequest{UserId: user})
	require.NoError(t, err)

	active := profile.GetUser().GetAvatarFilename()
	_, err = storage.GetURL(ctx, active)
	require.NoError(t, err, "current avatar must survive cleanup")
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM file WHERE created_by=$1`, user).Scan(&count))
	require.Equal(t, 1, count, "parallel replacements leave only the current metadata")

	_, err = users.DeleteAvatar(ctx, &pb.DeleteAvatarRequest{UserId: user})
	require.NoError(t, err)
	drain()

	_, err = storage.GetURL(ctx, active)
	require.ErrorIs(t, err, shareds3.ErrFileNotFound)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM file WHERE created_by=$1`, user).Scan(&count))
	require.Zero(t, count)

	log := testutil.DiscardLogger()
	films := postgres.NewFilmRepo(db, log)
	film, err := films.CreateFilm(ctx, domain.CreateFilmInput{
		Title:       "Cleanup test",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 2026,
		AgeLimit:    0,
	})
	require.NoError(t, err)
	repo, err := postgres.NewPosterRepo(db, log)
	require.NoError(t, err)
	objects, err := posters3.NewPosterStorage(storage)
	require.NoError(t, err)
	posters, err := usecase.NewPosterUseCase(repo, films, objects, usecase.PosterOptions{
		MaxSize:        5 << 20,
		MaxDimension:   4096,
		CleanupTimeout: 5 * time.Second,
	}, log)
	require.NoError(t, err)
	var previous string

	for range 2 {
		key, err := posters.UploadPoster(ctx, usecase.UploadPosterInput{
			UserID: domain.UserID(user),
			FilmID: film,
			Body:   bytes.NewReader(imageData.Bytes()),
			Size:   int64(imageData.Len()),
		})
		require.NoError(t, err)
		keys <- key

		drain()

		if previous != "" {
			_, err = storage.GetURL(ctx, previous)
			require.ErrorIs(t, err, shareds3.ErrFileNotFound)
		}

		_, err = storage.GetURL(ctx, key)
		require.NoError(t, err)

		previous = key
	}

	require.NoError(t, posters.DeletePoster(ctx, film))
	drain()

	_, err = storage.GetURL(ctx, previous)
	require.ErrorIs(t, err, shareds3.ErrFileNotFound)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM file WHERE created_by=$1`, user).Scan(&count))
	require.Zero(t, count)
}
