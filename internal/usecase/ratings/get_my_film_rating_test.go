package usecase

import (
	"context"
	"errors"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"testing"
)

type personalRatingStub struct {
	score *int
	err   error
	calls int
}

func (r *personalRatingStub) GetMyScore(_ context.Context, user domain.UserID, film domain.FilmID) (*int, error) {
	r.calls++
	if user != 7 || film != 42 {
		panic("wrong account or film")
	}
	return r.score, r.err
}

type ratingFilmStub struct{ err error }

func (r ratingFilmStub) GetFilmByID(context.Context, domain.FilmID) (domain.Film, error) {
	return domain.Film{}, r.err
}

type personalRatingLog struct{}

func (personalRatingLog) Info(string, map[string]any)  {}
func (personalRatingLog) Debug(string, map[string]any) {}
func (personalRatingLog) Warn(string, map[string]any)  {}
func (personalRatingLog) Error(string, map[string]any) {}

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
			repo := &personalRatingStub{score: tc.score, err: tc.repoErr}
			uc, err := NewGetMyFilmRatingUseCase(repo, ratingFilmStub{tc.filmErr}, personalRatingLog{})
			if err != nil {
				t.Fatal(err)
			}
			out, err := uc.Run(context.Background(), GetMyFilmRatingInput{AccountID: tc.user, FilmID: 42})
			if !errors.Is(err, tc.wantErr) || repo.calls != tc.calls {
				t.Fatalf("error=%v, calls=%d", err, repo.calls)
			}
			if err == nil && out.Score != tc.score {
				t.Fatal("rating changed")
			}
		})
	}
}
