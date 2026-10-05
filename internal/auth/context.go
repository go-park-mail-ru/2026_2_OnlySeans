package auth

import "context"

type userIDCtxKey struct{}

func ContextWithUserID(ctx context.Context, id UserID) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

func UserIDFromContext(ctx context.Context) (UserID, bool) {
	id, ok := ctx.Value(userIDCtxKey{}).(UserID)
	return id, ok
}
