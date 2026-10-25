package domain

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	MinReviewTitleLen   = 3
	MaxReviewTitleLen   = 255
	MinReviewContentLen = 50
	MaxReviewContentLen = 20000
)

type ReviewID int64

type Review struct {
	ID               ReviewID  `db:"id"`
	AuthorID         UserID    `db:"account_id"`
	FilmID           FilmID    `db:"film_id"`
	Title            string    `db:"title"`
	Content          string    `db:"content"`
	ContainsSpoilers bool      `db:"contains_spoilers"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// ReviewInput — то, что автор задаёт сам при создании и изменении отзыва.
type ReviewInput struct {
	Title            string
	Content          string
	ContainsSpoilers bool
}

// Validate проверяет длину заголовка и текста в символах, а не в байтах.
func (in ReviewInput) Validate() error {
	titleLen := utf8.RuneCountInString(in.Title)
	if titleLen < MinReviewTitleLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidTitle, MinReviewTitleLen)
	}
	if titleLen > MaxReviewTitleLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidTitle, MaxReviewTitleLen)
	}

	contentLen := utf8.RuneCountInString(in.Content)
	if contentLen < MinReviewContentLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidContent, MinReviewContentLen)
	}
	if contentLen > MaxReviewContentLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidContent, MaxReviewContentLen)
	}
	return nil
}

type ReviewRepository interface {
	Create(ctx context.Context, authorID UserID, filmID FilmID, input ReviewInput) (Review, error)
	GetByID(ctx context.Context, id ReviewID) (Review, error)
	ListByFilm(ctx context.Context, filmID FilmID, limit, offset int) ([]Review, int, error)
	Update(ctx context.Context, id ReviewID, input ReviewInput) (Review, error)
	Delete(ctx context.Context, id ReviewID, deletedBy UserID) error
}
