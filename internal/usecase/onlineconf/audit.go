package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListAuditUseCase struct {
	repository oc.Repository
	appLogger  logger.Logger
}

func NewListAuditUseCase(repository oc.Repository, appLogger logger.Logger) (*ListAuditUseCase, error) {
	if repository == nil {
		return nil, oc.ErrNilRepository
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListAuditUseCase{repository: repository, appLogger: appLogger}, nil
}

func (uc *ListAuditUseCase) Run(ctx context.Context, input oc.AuditFilter) (*ListOutput[oc.AuditEntry], error) {
	switch input.ResourceKind {
	case "", oc.ResourceScope, oc.ResourceConfig, oc.ResourceFlag:
	default:
		return nil, oc.ErrInvalidAuditFilter
	}

	if input.ResourceID < 0 || input.ResourceID > oc.MaxSafeID {
		return nil, oc.ErrInvalidAuditFilter
	}

	options, err := prepareList(oc.ListOptions{Limit: input.Limit, Offset: input.Offset})
	if err != nil {
		return nil, err
	}

	input.Limit = options.Limit
	input.Offset = options.Offset

	items, total, err := uc.repository.ListAudit(ctx, input)
	if err != nil {
		uc.appLogger.ErrorContext(ctx, "onlineconf audit read failed", nil)
		return nil, fmt.Errorf("list onlineconf audit: %w", err)
	}

	if items == nil {
		items = make([]oc.AuditEntry, 0)
	}

	return &ListOutput[oc.AuditEntry]{Items: items, Total: total}, nil
}
