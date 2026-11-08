package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

const registrationTestHash = "private-test-password-hash"

type registrationHasher struct{ err error }

func (h registrationHasher) Hash(string) (string, error) {
	return registrationTestHash, h.err
}

func TestRegistrationStageLogs(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	tests := []struct {
		name         string
		hashErr      error
		createErr    error
		getErr       error
		sessionErr   error
		invalid      bool
		wantMessages []string
	}{
		{
			name:         "success",
			wantMessages: []string{"registration started", "user created", "session created", "user registered"},
		},
		{
			name:         "invalid input",
			invalid:      true,
			wantMessages: []string{"registration started", "registration rejected: invalid input"},
		},
		{
			name:         "hash failure",
			hashErr:      storageErr,
			wantMessages: []string{"registration started", "password hashing failed"},
		},
		{
			name:         "creation failure",
			createErr:    storageErr,
			wantMessages: []string{"registration started", "user creation failed"},
		},
		{
			name:         "read after creation failure",
			getErr:       storageErr,
			wantMessages: []string{"registration started", "user created", "get created user failed"},
		},
		{
			name:         "session failure after creation",
			sessionErr:   storageErr,
			wantMessages: []string{"registration started", "user created", "session creation failed"},
		},
	}
	for _, tt := range tests {
		for _, level := range []logger.LogLevelType{logger.LogLevelInfo, logger.LogLevelDebug} {
			t.Run(tt.name+"/"+string(level), func(t *testing.T) {
				var output bytes.Buffer
				log, err := logger.NewLogger(&output, level, "json")
				require.NoError(t, err)
				repo := flakyRepo{InMemoryUserRepo: authservice.NewInMemoryUserRepo(), createErr: tt.createErr, getUserErr: tt.getErr}
				sessions := flakyStore{InMemorySessionStore: authservice.NewInMemorySessionStore(time.Hour), createErr: tt.sessionErr}
				uc, err := authusecase.NewRegisterUserUseCase(repo, sessions, registrationHasher{err: tt.hashErr}, log)
				require.NoError(t, err)
				input := authusecase.NewRegisterUserInput("private@example.com", "logtest", "PrivatePassword123")
				if tt.invalid {
					input.Password = "Short1"
				}
				result, err := uc.Run(t.Context(), input)
				failed := tt.invalid || tt.hashErr != nil || tt.createErr != nil || tt.getErr != nil || tt.sessionErr != nil
				require.Equal(t, failed, err != nil)
				raw := output.String()
				require.NotContains(t, raw, input.Password)
				require.NotContains(t, raw, registrationTestHash)
				require.NotContains(t, raw, input.Email)
				if result != nil {
					require.NotContains(t, raw, result.Session.ID)
				}
				decoder := json.NewDecoder(bytes.NewBufferString(raw))
				var messages, debugMessages []string
				for decoder.More() {
					var entry map[string]any
					require.NoError(t, decoder.Decode(&entry))
					message := entry["msg"].(string)
					if entry["level"] == "DEBUG" {
						debugMessages = append(debugMessages, message)
						continue
					}
					messages = append(messages, message)
					if message == "user created" || message == "session created" || message == "user registered" {
						require.Equal(t, "INFO", entry["level"])
						require.Equal(t, float64(1), entry["user_id"])
					}
					if message == "session created" {
						require.NotEmpty(t, entry["expires_at"])
					}
					if message == "session creation failed" || message == "get created user failed" {
						require.Equal(t, "ERROR", entry["level"])
						require.Equal(t, float64(1), entry["user_id"])
					}
				}
				require.Equal(t, tt.wantMessages, messages)
				if level == logger.LogLevelDebug && !failed {
					require.Equal(t, []string{"registration input validated", "password hashed", "creating user", "created user loaded"}, debugMessages)
				}
				if level == logger.LogLevelInfo {
					require.Empty(t, debugMessages)
				}
				if tt.invalid {
					require.ErrorIs(t, err, domain.ErrInvalidPassword)
				}
			})
		}
	}
}
