package domain

import "context"

type PosterFile struct {
	ExternalURL *string
	Filename    string
	ContentType string
	Size        int64
}

type PosterRepository interface {
	GetPoster(ctx context.Context, filmID FilmID) (*PosterFile, error)
	DeletePoster(ctx context.Context, filmID FilmID) error
	SavePoster(ctx context.Context, filmID FilmID, userID UserID, poster PosterFile) error
}
