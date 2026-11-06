package auth_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHandlerCanEditPersons(t *testing.T) {
	h, users, _ := newTestAuth(t)
	ctx := context.Background()
	id, err := users.Create(ctx, "editor@example.com", "editor", "hash")
	require.NoError(t, err)
	user, err := users.GetUser(ctx, id)
	require.NoError(t, err)
	tests := []struct {
		name    string
		role    domain.Role
		allowed bool
	}{
		{name: "reader", role: domain.RoleReader},
		{name: "editor", role: domain.RoleEditor, allowed: true},
		{name: "admin", role: domain.RoleAdmin, allowed: true},
		{name: "unknown role", role: "other"},
		{name: "missing role"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The same account is checked again after its role changes.
			user.Role = tt.role
			allowed, err := h.CanEditPersons(ctx, id)
			require.NoError(t, err)
			require.Equal(t, tt.allowed, allowed)
		})
	}
}
