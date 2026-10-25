package auth

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type userIDCtxKey struct{}

func ContextWithUserID(ctx context.Context, id domain.UserID) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	id, ok := ctx.Value(userIDCtxKey{}).(domain.UserID)
	return id, ok
}
