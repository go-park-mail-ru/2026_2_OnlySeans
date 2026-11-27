package grpc_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/tests/mocks"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

func TestRefreshSession(t *testing.T) {
	session := &domain.Session{ID: uuid.New(), UserID: 7, Role: "reader", ExpiresAt: time.Now().Add(time.Hour)}
	for _, tc := range []struct {
		name    string
		failure error
		code    codes.Code
		result  *domain.Session
	}{
		{name: "success", code: codes.OK, result: session},
		{name: "missing", failure: domain.ErrSessionNotFound, code: codes.Unauthenticated},
		{name: "expired", failure: domain.ErrSessionExpired, code: codes.Unauthenticated},
		{name: "storage error", failure: errors.New("private failure"), code: codes.Internal},
		{name: "nil result", code: codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := mocks.NewMockAuthUseCase(gomock.NewController(t))
			uc.EXPECT().RefreshSession(t.Context(), session.ID).Return(tc.result, tc.failure)
			h := authgrpc.NewAuthHandler(uc)
			result, err := h.RefreshSession(t.Context(), &pb.RefreshSessionRequest{SessionId: session.ID.String()})
			require.Equal(t, tc.code, status.Code(err))

			if tc.code == codes.OK {
				require.Equal(t, session.ID.String(), result.Session.Id)
				require.Equal(t, session.ExpiresAt.UTC(), result.Session.ExpiresAt.AsTime())
			} else {
				require.Nil(t, result)
			}
		})
	}

	h := authgrpc.NewAuthHandler(nil)
	for _, req := range []*pb.RefreshSessionRequest{nil, {}, {SessionId: "invalid"}} {
		_, err := h.RefreshSession(t.Context(), req)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}
