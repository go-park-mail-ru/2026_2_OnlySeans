package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	createConfigOperation = "onlineconf.config.create"
	updateConfigOperation = "onlineconf.config.update"
	deleteConfigOperation = "onlineconf.config.delete"
)

type ConfigsUseCase struct{ management }

func NewConfigsUseCase(repository oc.Repository, refresher SnapshotRefresher, appLogger logger.Logger) (*ConfigsUseCase, error) {
	dependencies, err := newManagement(repository, refresher, appLogger)
	if err != nil {
		return nil, err
	}

	return &ConfigsUseCase{management: dependencies}, nil
}

func (uc *ConfigsUseCase) Create(ctx context.Context, input CreateConfigInput) (*oc.Config, error) {
	return mutateResource(ctx, &uc.management, input.ActorID, createConfigOperation, func() (*oc.Config, error) {
		value, err := prepareConfig(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate config: %w", err)
		}

		created, err := uc.repository.CreateConfig(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("create config: %w", err)
		}

		return created, nil
	})
}

func (uc *ConfigsUseCase) Get(ctx context.Context, input GetResourceInput) (*oc.Config, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	result, err := uc.repository.GetConfig(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return result, nil
}

func (uc *ConfigsUseCase) List(ctx context.Context, input oc.ListOptions) (*ListOutput[oc.Config], error) {
	options, err := prepareList(input)
	if err != nil {
		return nil, err
	}

	items, total, err := uc.repository.ListConfigs(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list configs: %w", err)
	}

	if items == nil {
		items = make([]oc.Config, 0)
	}

	return &ListOutput[oc.Config]{Items: items, Total: total}, nil
}

func (uc *ConfigsUseCase) Update(ctx context.Context, input UpdateConfigInput) (*oc.Config, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	if err := validateVersion(input.Version); err != nil {
		return nil, err
	}

	return mutateResource(ctx, &uc.management, input.ActorID, updateConfigOperation, func() (*oc.Config, error) {
		value, err := prepareConfig(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate config: %w", err)
		}

		value.ID = input.ID
		value.Version = input.Version

		updated, err := uc.repository.UpdateConfig(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("update config: %w", err)
		}

		return updated, nil
	})
}

func (uc *ConfigsUseCase) Delete(ctx context.Context, input DeleteResourceInput) error {
	if err := validateID(input.ID); err != nil {
		return err
	}

	if err := validateVersion(input.Version); err != nil {
		return err
	}

	if _, err := mutateResource(ctx, &uc.management, input.ActorID, deleteConfigOperation, func() (*oc.Config, error) {
		if err := uc.repository.DeleteConfig(ctx, input.ActorID, input.ID, input.Version); err != nil {
			return nil, fmt.Errorf("delete config: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}

func prepareConfig(input ConfigFields) (oc.Config, error) {
	value := oc.Config{Scope: input.Scope, Key: input.Key, Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description), Value: append(json.RawMessage(nil), input.Value...)}
	if err := value.Validate(); err != nil {
		return value, err
	}

	return value, nil
}
