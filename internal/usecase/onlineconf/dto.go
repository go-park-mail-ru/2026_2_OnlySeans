package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const initialVersion = 1

type SnapshotRefresher interface{ RequestRefresh() }

type ScopeFields struct {
	Name        string
	Title       string
	Description string
}
type ConfigFields struct {
	Scope       string
	Key         string
	Title       string
	Description string
	Value       json.RawMessage
}
type FlagFields struct {
	Scope           string
	Key             string
	Title           string
	Description     string
	Enabled         bool
	RolloutPercent  int
	ForceOnUserIDs  []domain.UserID
	ForceOffUserIDs []domain.UserID
}

type GetResourceInput struct{ ID int64 }
type DeleteResourceInput struct {
	ActorID domain.UserID
	ID      int64
	Version int64
}
type ListOutput[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type management struct {
	repository oc.Repository
	refresher  SnapshotRefresher
	appLogger  logger.Logger
}

func newManagement(repository oc.Repository, refresher SnapshotRefresher, appLogger logger.Logger) (management, error) {
	if repository == nil {
		return management{}, oc.ErrNilRepository
	}

	if refresher == nil {
		return management{}, oc.ErrNilSnapshotRefresher
	}

	if appLogger == nil {
		return management{}, domain.ErrInvalidAppLogger
	}

	return management{repository: repository, refresher: refresher, appLogger: appLogger}, nil
}

func validateID(id int64) error {
	if id < oc.MinUserID || id > oc.MaxSafeID {
		return oc.ErrInvalidID
	}

	return nil
}

func validateActor(id domain.UserID) error {
	if id < oc.MinUserID || int64(id) > oc.MaxSafeID {
		return oc.ErrInvalidUserID
	}

	return nil
}

func validateVersion(version int64) error {
	if version < initialVersion {
		return oc.ErrInvalidVersion
	}

	return nil
}

func prepareList(input oc.ListOptions) (oc.ListOptions, error) {
	if input.Limit == 0 {
		input.Limit = oc.DefaultListLimit
	}

	if input.Limit < 0 || input.Limit > oc.MaxListLimit || input.Offset < 0 {
		return input, oc.ErrInvalidPagination
	}

	if input.Scope != "" {
		if err := oc.ValidateKey(input.Scope); err != nil {
			return input, fmt.Errorf("validate scope filter: %w", err)
		}
	}

	return input, nil
}

func mutateResource[T any](ctx context.Context, uc *management, actorID domain.UserID, operation string, change func() (*T, error)) (*T, error) {
	if err := validateActor(actorID); err != nil {
		return nil, err
	}

	fields := map[string]any{"actor_id": actorID, "operation": operation}
	uc.appLogger.DebugContext(ctx, "onlineconf mutation started", fields)

	result, err := change()
	if err != nil {
		uc.appLogger.WarnContext(ctx, "onlineconf mutation failed", fields)
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	uc.refresher.RequestRefresh()
	uc.appLogger.InfoContext(ctx, "onlineconf mutation committed", fields)

	return result, nil
}

type CreateScopeInput struct {
	ActorID domain.UserID
	Fields  ScopeFields
}
type UpdateScopeInput struct {
	ActorID domain.UserID
	ID      int64
	Version int64
	Fields  ScopeFields
}

type CreateConfigInput struct {
	ActorID domain.UserID
	Fields  ConfigFields
}
type UpdateConfigInput struct {
	ActorID domain.UserID
	ID      int64
	Version int64
	Fields  ConfigFields
}

type CreateFlagInput struct {
	ActorID domain.UserID
	Fields  FlagFields
}
type UpdateFlagInput struct {
	ActorID domain.UserID
	ID      int64
	Version int64
	Fields  FlagFields
}
