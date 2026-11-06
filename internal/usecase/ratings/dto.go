package usecase

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type SetFilmRatingInput struct {
	AccountID domain.UserID
	FilmID    domain.FilmID
	Score     int
}

func NewSetFilmRatingInput(accountID domain.UserID, filmID domain.FilmID, score int) *SetFilmRatingInput {
	return &SetFilmRatingInput{
		AccountID: accountID,
		FilmID:    filmID,
		Score:     score}
}

type GetFilmRatingInput struct {
	FilmID domain.FilmID
}

func NewGetFilmRatingInput(filmID domain.FilmID) *GetFilmRatingInput {
	return &GetFilmRatingInput{FilmID: filmID}
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
	Limit        int
	Offset       int
}

func NewGetCollectionRatingsInput(collectionID domain.CollectionID, limit, offset int) *GetCollectionRatingsInput {
	return &GetCollectionRatingsInput{CollectionID: collectionID, Limit: limit, Offset: offset}
}

type GetCollectionRatingsOutput struct {
	Ratings []domain.FilmRatingSummary
	Total   int
}

func NewGetCollectionRatingsOutput(ratings []domain.FilmRatingSummary, total int) *GetCollectionRatingsOutput {
	return &GetCollectionRatingsOutput{Ratings: ratings, Total: total}
}

type DeleteFilmRatingInput struct {
	AccountID domain.UserID
	FilmID    domain.FilmID
}

func NewDeleteFilmRatingInput(accountID domain.UserID, filmID domain.FilmID) *DeleteFilmRatingInput {
	return &DeleteFilmRatingInput{AccountID: accountID, FilmID: filmID}
}
