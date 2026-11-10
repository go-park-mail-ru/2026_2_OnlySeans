package ratings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	ratingsuc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetMyFilmRating(t *testing.T) {
	score := 8
	storageErr := errors.New("storage unavailable")
	for _, tc := range []struct {
		name                      string
		user                      domain.UserID
		score                     *int
		filmErr, repoErr, wantErr error
		calls                     int
	}{
		{"saved", 7, &score, nil, nil, nil, 1},
		{"no rating", 7, nil, nil, nil, nil, 1},
		{"unauthorized", 0, nil, nil, nil, domain.ErrUnauthorized, 0},
		{"missing film", 7, nil, domain.ErrFilmNotFound, nil, domain.ErrFilmNotFound, 0},
		{"lookup failure", 7, nil, storageErr, nil, storageErr, 0},
		{"rating failure", 7, nil, nil, storageErr, storageErr, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockFilmRatingRepository(ctrl)
			films := mocks.NewMockFilmRepository(ctrl)
			if tc.user > 0 {
				lookup := films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(42)).Return(domain.Film{}, tc.filmErr)
				if tc.calls > 0 {
					repo.EXPECT().GetMyScore(gomock.Any(), tc.user, domain.FilmID(42)).Return(tc.score, tc.repoErr).After(lookup)
				}
			}
			uc, err := ratingsuc.NewGetMyFilmRatingUseCase(repo, films, testLogger(t))
			if err != nil {
				t.Fatal(err)
			}
			out, err := uc.Run(context.Background(), ratingsuc.GetMyFilmRatingInput{AccountID: tc.user, FilmID: 42})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v", err)
			}
			if err == nil && out.Score != tc.score {
				t.Fatal("rating changed")
			}
		})
	}
}
