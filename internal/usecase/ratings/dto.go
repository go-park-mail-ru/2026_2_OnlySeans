package usecase

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type SetFilmRatingInput struct {
	AccountID domain.UserID
	FilmID    domain.FilmID
	Score     int16
}

func NewSetFilmRatingInput(accountID domain.UserID, filmID domain.FilmID, score int16) SetFilmRatingInput {
	return SetFilmRatingInput{AccountID: accountID, FilmID: filmID, Score: score}
}

type GetFilmRatingInput struct {
	FilmID domain.FilmID
}

func NewGetFilmRatingInput(filmID domain.FilmID) GetFilmRatingInput {
	return GetFilmRatingInput{FilmID: filmID}
}

type SetFilmRatingOutput struct {
	Rating domain.FilmRating
}

func NewSetFilmRatingOutput(rating domain.FilmRating) *SetFilmRatingOutput {
	return &SetFilmRatingOutput{Rating: rating}
}

type GetFilmRatingOutput struct {
	Summary domain.FilmRatingSummary
}

func NewGetFilmRatingOutput(summary domain.FilmRatingSummary) *GetFilmRatingOutput {
	return &GetFilmRatingOutput{Summary: summary}
}

type GetCollectionRatingsInput struct {
	CollectionID domain.CollectionID
}

func NewGetCollectionRatingsInput(collectionID domain.CollectionID) GetCollectionRatingsInput {
	return GetCollectionRatingsInput{CollectionID: collectionID}
}

type GetCollectionRatingsOutput struct {
	Ratings []domain.FilmRatingSummary
}

func NewGetCollectionRatingsOutput(ratings []domain.FilmRatingSummary) *GetCollectionRatingsOutput {
	return &GetCollectionRatingsOutput{Ratings: ratings}
}

type DeleteFilmRatingInput struct {
	AccountID domain.UserID
	FilmID    domain.FilmID
}

func NewDeleteFilmRatingInput(accountID domain.UserID, filmID domain.FilmID) DeleteFilmRatingInput {
	return DeleteFilmRatingInput{AccountID: accountID, FilmID: filmID}
}
