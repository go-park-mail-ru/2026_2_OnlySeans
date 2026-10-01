package auth

import "context"

type userIDCtxKey struct{}

// ContextWithUserID кладёт ID пользователя в контекст (вызывает мидлварь).
func ContextWithUserID(ctx context.Context, id UserID) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

// UserIDFromContext достаёт ID пользователя, положенный мидлварью.
func UserIDFromContext(ctx context.Context) (UserID, bool) {
	id, ok := ctx.Value(userIDCtxKey{}).(UserID)
	return id, ok
}
