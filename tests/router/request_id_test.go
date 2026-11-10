package router_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authrepo "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	authuc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestAccessUsecaseAndDatabaseLogsShareRequestID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	var buffer bytes.Buffer
	appLogger, err := logger.NewLogger(&buffer, logger.LogLevelDebug, "json")
	require.NoError(t, err)
	uc, err := authuc.NewGetUserUseCase(authrepo.NewPostgresUserRepo(db, appLogger), appLogger)
	require.NoError(t, err)
	handler := router.WithRequestID(router.WithLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := uc.Run(r.Context(), authuc.GetUserInput{UserID: 7})
		if err != nil {
			appLogger.WarnContext(r.Context(), "handler failed", nil)
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusNotFound, "user not found", appLogger)
			return
		}
		appLogger.InfoContext(r.Context(), "handler completed", nil)
		httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, user, appLogger)
	}), appLogger))
	var previous string
	for _, found := range []bool{true, false} {
		buffer.Reset()
		query := mock.ExpectQuery("FROM account").WithArgs(domain.UserID(7))
		if found {
			query.WillReturnRows(sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "role_code", "avatar_filename"}).AddRow(7, "anna@example.com", "Anna", "hash", "reader", nil))
		} else {
			query.WillReturnError(sql.ErrNoRows)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
		if found {
			require.Equal(t, http.StatusOK, rec.Code)
		} else {
			require.Equal(t, http.StatusNotFound, rec.Code)
		}
		id := rec.Header().Get("X-Request-ID")
		require.NotEmpty(t, id)
		require.NotEqual(t, previous, id)
		previous = id
		decoder := json.NewDecoder(&buffer)
		messages := map[string]bool{}
		sawDatabase := false
		for {
			var record map[string]any
			err := decoder.Decode(&record)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			require.Equal(t, id, record["request_id"], record)
			messages[record["msg"].(string)] = true
			if record["operation"] == "account.get_user" {
				sawDatabase = true
			}
		}
		require.True(t, messages["request started"])
		require.True(t, messages["get user started"])
		require.True(t, messages["request handled"])
		require.True(t, sawDatabase)
		if found {
			require.True(t, messages["user retrieved"])
			require.True(t, messages["handler completed"])
		} else {
			require.True(t, messages["get user failed"])
			require.True(t, messages["handler failed"])
		}
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
