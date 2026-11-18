package middleware

import (
	"context"
)

type userIDCtxKey struct{}

func ContextWithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDCtxKey{}).(int64)
	return id, ok
}

type userRoleCtxKey struct{}

func ContextWithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, userRoleCtxKey{}, role)
}

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(userRoleCtxKey{}).(string)
	return role, ok
}
