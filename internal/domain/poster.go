package domain

import "context"

type PosterFile struct {
	Filename    string
	ContentType string
	Size        int64
}

type PosterRepository interface {
	SavePoster(ctx context.Context, filmID FilmID, userID UserID, poster PosterFile) error
}
