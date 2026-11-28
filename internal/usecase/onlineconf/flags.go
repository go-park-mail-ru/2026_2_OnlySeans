package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	createFlagOperation = "onlineconf.flag.create"
	updateFlagOperation = "onlineconf.flag.update"
	deleteFlagOperation = "onlineconf.flag.delete"
)

type FlagsUseCase struct{ management }

func NewFlagsUseCase(repository oc.Repository, refresher SnapshotRefresher, appLogger logger.Logger) (*FlagsUseCase, error) {
	dependencies, err := newManagement(repository, refresher, appLogger)
	if err != nil {
		return nil, err
	}

	return &FlagsUseCase{management: dependencies}, nil
}

func (uc *FlagsUseCase) Create(ctx context.Context, input CreateFlagInput) (*oc.FeatureFlag, error) {
	return mutateResource(ctx, &uc.management, input.ActorID, createFlagOperation, func() (*oc.FeatureFlag, error) {
		value, err := prepareFlag(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate flag: %w", err)
		}

		created, err := uc.repository.CreateFlag(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("create flag: %w", err)
		}

		return created, nil
	})
}

func (uc *FlagsUseCase) Get(ctx context.Context, input GetResourceInput) (*oc.FeatureFlag, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	result, err := uc.repository.GetFlag(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}

	return result, nil
}

func (uc *FlagsUseCase) List(ctx context.Context, input oc.ListOptions) (*ListOutput[oc.FeatureFlag], error) {
	options, err := prepareList(input)
	if err != nil {
		return nil, err
	}

	items, total, err := uc.repository.ListFlags(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}

	if items == nil {
		items = make([]oc.FeatureFlag, 0)
	}

	return &ListOutput[oc.FeatureFlag]{Items: items, Total: total}, nil
}

func (uc *FlagsUseCase) Update(ctx context.Context, input UpdateFlagInput) (*oc.FeatureFlag, error) {
	if err := validateID(input.ID); err != nil {
		return nil, err
	}

	if err := validateVersion(input.Version); err != nil {
		return nil, err
	}

	return mutateResource(ctx, &uc.management, input.ActorID, updateFlagOperation, func() (*oc.FeatureFlag, error) {
		value, err := prepareFlag(input.Fields)
		if err != nil {
			return nil, fmt.Errorf("validate flag: %w", err)
		}

		value.ID = input.ID
		value.Version = input.Version

		updated, err := uc.repository.UpdateFlag(ctx, input.ActorID, value)
		if err != nil {
			return nil, fmt.Errorf("update flag: %w", err)
		}

		return updated, nil
	})
}

func (uc *FlagsUseCase) Delete(ctx context.Context, input DeleteResourceInput) error {
	if err := validateID(input.ID); err != nil {
		return err
	}

	if err := validateVersion(input.Version); err != nil {
		return err
	}

	if _, err := mutateResource(ctx, &uc.management, input.ActorID, deleteFlagOperation, func() (*oc.FeatureFlag, error) {
		if err := uc.repository.DeleteFlag(ctx, input.ActorID, input.ID, input.Version); err != nil {
			return nil, fmt.Errorf("delete flag: %w", err)
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}

func prepareFlag(input FlagFields) (oc.FeatureFlag, error) {
	value := oc.FeatureFlag{Scope: input.Scope, Key: input.Key, Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description), Enabled: input.Enabled, RolloutPercent: input.RolloutPercent, ForceOnUserIDs: append([]domain.UserID{}, input.ForceOnUserIDs...), ForceOffUserIDs: append([]domain.UserID{}, input.ForceOffUserIDs...)}
	if err := oc.ValidateKey(value.Scope); err != nil {
		return value, err
	}

	if err := oc.ValidateKey(value.Key); err != nil {
		return value, err
	}

	if value.Title == "" {
		return value, oc.ErrInvalidTitle
	}

	if err := oc.ValidateRollout(value.RolloutPercent); err != nil {
		return value, err
	}

	overrides := make([]oc.Override, 0, len(value.ForceOnUserIDs)+len(value.ForceOffUserIDs))
	for _, id := range value.ForceOnUserIDs {
		overrides = append(overrides, oc.Override{UserID: id, Enabled: true})
	}

	for _, id := range value.ForceOffUserIDs {
		overrides = append(overrides, oc.Override{UserID: id, Enabled: false})
	}

	if err := oc.ValidateOverrides(overrides); err != nil {
		return value, err
	}

	return value, nil
}
