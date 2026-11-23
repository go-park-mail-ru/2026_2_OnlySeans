package reviews

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

// Operations describes the reviews capabilities required by the HTTP gateway.
// The gRPC client implements these functions without exposing service internals.
type Operations struct {
	List   func(context.Context, domain.FilmID, int, int) ([]domain.Review, int, error)
	Create func(context.Context, domain.UserID, domain.FilmID, domain.ReviewInput) (domain.ReviewID, error)
	Update func(context.Context, domain.UserID, domain.ReviewID, domain.ReviewInput, int) (domain.Review, error)
	Delete func(context.Context, domain.UserID, domain.ReviewID) error
}
