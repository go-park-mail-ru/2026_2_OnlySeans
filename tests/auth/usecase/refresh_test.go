package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/tests/mocks"
)

func TestAuthRefresh(t *testing.T) {
	for _, failure := range []error{nil, domain.ErrSessionNotFound, domain.ErrSessionExpired, errors.New("storage failed")} {
		sessions := mocks.NewMockSessionUseCase(gomock.NewController(t))
		session := domain.Session{ID: uuid.New(), UserID: 7, Role: "reader", ExpiresAt: time.Now().Add(time.Hour)}
		sessions.EXPECT().Refresh(t.Context(), session.ID).Return(session, failure)
		uc := usecase.NewAuthUseCase(nil, sessions, nil)
		result, err := uc.RefreshSession(t.Context(), session.ID)
		require.ErrorIs(t, err, failure)

		if failure != nil {
			require.Nil(t, result)
		} else {
			require.Equal(t, &session, result)
		}
	}
}
