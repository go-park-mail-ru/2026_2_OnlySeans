package usecase

import (
	"context"
	"fmt"
	"strings"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	createScopeOperation = "onlineconf.scope.create"
	updateScopeOperation = "onlineconf.scope.update"
	deleteScopeOperation = "onlineconf.scope.delete"
)

type ScopesUseCase struct{ management }

func NewScopesUseCase(repository oc.Repository, refresher SnapshotRefresher, appLogger logger.Logger) (*ScopesUseCase, error) {
	dependencies, err := newManagement(repository, refresher, appLogger)
	if err != nil {
		return nil, err
	}

	return &ScopesUseCase{management: dependencies}, nil
}

func (uc *ScopesUseCase) Create(ctx context.Context, input CreateScopeInput) (*oc.Scope, error) {
	return mutateResource(ctx, &uc.management, input.ActorID, createScopeOperation, func() (*oc.Scope, error) {
		value, err := prepareScope(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate scope: %w", err)
		}

		created, err := uc.repository.CreateScope(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("create scope: %w", err)
		}

		return created, nil
	})
}

func (uc *ScopesUseCase) Get(ctx context.Context, input GetResourceInput) (*oc.Scope, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	result, err := uc.repository.GetScope(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get scope: %w", err)
	}

	return result, nil
}

func (uc *ScopesUseCase) List(ctx context.Context, input oc.ListOptions) (*ListOutput[oc.Scope], error) {
	options, err := prepareList(input)
	if err != nil {
		return nil, err
	}

	items, total, err := uc.repository.ListScopes(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list scopes: %w", err)
	}

	if items == nil {
		items = make([]oc.Scope, 0)
	}

	return &ListOutput[oc.Scope]{Items: items, Total: total}, nil
}

func (uc *ScopesUseCase) Update(ctx context.Context, input UpdateScopeInput) (*oc.Scope, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	if err := validateVersion(input.Version); err != nil {
		return nil, err
	}

	return mutateResource(ctx, &uc.management, input.ActorID, updateScopeOperation, func() (*oc.Scope, error) {
		value, err := prepareScope(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate scope: %w", err)
		}

		value.ID = input.ID
		value.Version = input.Version

		updated, err := uc.repository.UpdateScope(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("update scope: %w", err)
		}

		return updated, nil
	})
}

func (uc *ScopesUseCase) Delete(ctx context.Context, input DeleteResourceInput) error {
	if err := validateID(input.ID); err != nil {
		return err
	}

	if err := validateVersion(input.Version); err != nil {
		return err
	}

	if _, err := mutateResource(ctx, &uc.management, input.ActorID, deleteScopeOperation, func() (*oc.Scope, error) {
		if err := uc.repository.DeleteScope(ctx, input.ActorID, input.ID, input.Version); err != nil {
			return nil, fmt.Errorf("delete scope: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}

func prepareScope(input ScopeFields) (oc.Scope, error) {
	value := oc.Scope{Name: input.Name, Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description)}
	if err := value.Validate(); err != nil {
		return value, err
	}

	return value, nil
}
