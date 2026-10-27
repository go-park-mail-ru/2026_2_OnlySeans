package usecase

import (
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CreateReviewInput struct {
	AuthorID domain.UserID
	FilmID   domain.FilmID
	Review   domain.ReviewInput
}

func NewCreateReviewInput(authorID domain.UserID, filmID domain.FilmID, review domain.ReviewInput) CreateReviewInput {
	return CreateReviewInput{AuthorID: authorID, FilmID: filmID, Review: review}
}

type UpdateReviewInput struct {
	UserID   domain.UserID
	ReviewID domain.ReviewID
	Review   domain.ReviewInput
}

func NewUpdateReviewInput(userID domain.UserID, reviewID domain.ReviewID, review domain.ReviewInput) UpdateReviewInput {
	return UpdateReviewInput{UserID: userID, ReviewID: reviewID, Review: review}
}

type DeleteReviewInput struct {
	UserID   domain.UserID
	ReviewID domain.ReviewID
}

func NewDeleteReviewInput(userID domain.UserID, reviewID domain.ReviewID) DeleteReviewInput {
	return DeleteReviewInput{UserID: userID, ReviewID: reviewID}
}

type ListReviewsInput struct {
	FilmID domain.FilmID
	Limit  int
	Offset int
}

func NewListReviewsInput(filmID domain.FilmID, limit, offset int) ListReviewsInput {
	return ListReviewsInput{FilmID: filmID, Limit: limit, Offset: offset}
}

type ReviewOutput struct {
	Review domain.Review
}

func NewReviewOutput(review *domain.Review) *ReviewOutput {
	return &ReviewOutput{Review: *review}
}

type ListReviewsOutput struct {
	Reviews []domain.Review
	Total   int
}

func NewListReviewsOutput(reviews []domain.Review, total int) *ListReviewsOutput {
	return &ListReviewsOutput{Reviews: reviews, Total: total}
}

// обрезает пробелы по краям и проверяет длину.
func prepareReview(in domain.ReviewInput) (*domain.ReviewInput, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Content = strings.TrimSpace(in.Content)
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return &in, nil
}
