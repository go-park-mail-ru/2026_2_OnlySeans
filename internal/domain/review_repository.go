package domain

import "context"

type ReviewRepository interface {
	Create(ctx context.Context, authorID UserID, filmID FilmID, input ReviewInput) (*Review, error)
	GetByID(ctx context.Context, id ReviewID) (*Review, error)
	ListByFilm(ctx context.Context, filmID FilmID, limit, offset int) ([]Review, int, error)
	Update(ctx context.Context, id ReviewID, input ReviewInput) (*Review, error)
	Delete(ctx context.Context, id ReviewID, deletedBy UserID) error
}
