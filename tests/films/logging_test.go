package films_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

type filmLoggingRepository struct {
	*filmsservice.InMemoryService
	err error
}

func (r filmLoggingRepository) CreateFilm(context.Context, domain.CreateFilmInput) (domain.FilmID, error) {
	return 42, r.err
}

func (r filmLoggingRepository) UpdateFilm(context.Context, domain.FilmID, domain.CreateFilmInput) error {
	return r.err
}

func (r filmLoggingRepository) DeleteFilm(context.Context, domain.FilmID) error {
	return r.err
}

func TestFilmMutationLogs(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		repoErr error
		invalid bool
		level   string
		message string
	}{
		{name: "create success", action: "create", level: "INFO", message: "film created"},
		{name: "update success", action: "update", level: "INFO", message: "film updated"},
		{name: "delete success", action: "delete", level: "INFO", message: "film deleted"},
		{name: "create invalid input", action: "create", invalid: true, level: "WARN", message: "create film rejected: invalid input"},
		{name: "update invalid input", action: "update", invalid: true, level: "WARN", message: "update film rejected: invalid input"},
		{name: "create database failure", action: "create", repoErr: errors.New("database unavailable"), level: "ERROR", message: "create film failed"},
		{name: "update database failure", action: "update", repoErr: errors.New("database unavailable"), level: "ERROR", message: "update film failed"},
		{name: "delete database failure", action: "delete", repoErr: errors.New("database unavailable"), level: "ERROR", message: "delete film failed"},
		{name: "delete missing film", action: "delete", repoErr: domain.ErrFilmNotFound, level: "DEBUG", message: "delete film failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			appLogger, err := logger.NewLogger(&output, logger.LogLevelDebug, "json")
			require.NoError(t, err)
			repo := filmLoggingRepository{InMemoryService: filmsservice.NewInMemoryService(), err: tt.repoErr}
			input := domain.CreateFilmInput{Title: "Alien", FilmType: domain.FilmTypeMovie, ReleaseYear: 1979, AgeLimit: 16}
			if tt.invalid {
				input.Title = ""
			}
			switch tt.action {
			case "create":
				uc, err := filmsusecase.NewCreateFilmUseCase(repo, appLogger)
				require.NoError(t, err)
				_, err = uc.Run(t.Context(), input)
				require.Equal(t, tt.invalid || tt.repoErr != nil, err != nil)
			case "update":
				uc, err := filmsusecase.NewUpdateFilmUseCase(repo, appLogger)
				require.NoError(t, err)
				err = uc.Run(t.Context(), 42, input)
				require.Equal(t, tt.invalid || tt.repoErr != nil, err != nil)
			case "delete":
				uc, err := filmsusecase.NewDeleteFilmUseCase(repo, appLogger)
				require.NoError(t, err)
				require.ErrorIs(t, uc.Run(t.Context(), 42), tt.repoErr)
			}
			decoder := json.NewDecoder(&output)
			var entries []map[string]any
			for decoder.More() {
				var entry map[string]any
				require.NoError(t, decoder.Decode(&entry))
				entries = append(entries, entry)
			}
			require.Len(t, entries, 2)
			require.Equal(t, "DEBUG", entries[0]["level"])
			require.Equal(t, tt.level, entries[1]["level"])
			require.Equal(t, tt.message, entries[1]["msg"])
			if !tt.invalid && (tt.repoErr == nil || tt.action != "create") {
				require.Equal(t, float64(42), entries[1]["film_id"])
			}
		})
	}
}
