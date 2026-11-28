package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type UserReader interface {
	GetUser(context.Context, domain.UserID) (*domain.User, error)
}
type SnapshotProvider interface{ Current() (*oc.Snapshot, error) }
type FlagEvaluator func(*oc.FeatureFlag, domain.UserID) oc.Evaluation

type EvaluateFlagInput struct {
	FlagID int64
	UserID domain.UserID
}

type EvaluateFlagUseCase struct {
	users     UserReader
	snapshots SnapshotProvider
	evaluate  FlagEvaluator
	appLogger logger.Logger
}

func NewEvaluateFlagUseCase(users UserReader, snapshots SnapshotProvider, evaluate FlagEvaluator, appLogger logger.Logger) (*EvaluateFlagUseCase, error) {
	if users == nil {
		return nil, oc.ErrNilUserReader
	}

	if snapshots == nil {
		return nil, oc.ErrNilSnapshotProvider
	}

	if evaluate == nil {
		return nil, oc.ErrNilEvaluator
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &EvaluateFlagUseCase{users: users, snapshots: snapshots, evaluate: evaluate, appLogger: appLogger}, nil
}

func (uc *EvaluateFlagUseCase) Run(ctx context.Context, input EvaluateFlagInput) (*oc.Decision, error) {
	if err := validateID(input.FlagID); err != nil {
		return nil, err
	}

	if err := validateActor(input.UserID); err != nil {
		return nil, err
	}

	user, err := uc.users.GetUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for flag evaluation: %w", err)
	}

	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	snapshot, err := uc.snapshots.Current()
	if err != nil {
		return nil, fmt.Errorf("get evaluation snapshot: %w", err)
	}

	if snapshot == nil {
		return nil, oc.ErrSnapshotNotReady
	}

	for i := range snapshot.Flags {
		flag := &snapshot.Flags[i]
		if flag.ID != input.FlagID {
			continue
		}

		evaluation := uc.evaluate(flag, input.UserID)
		uc.appLogger.DebugContext(ctx, "onlineconf flag evaluated", map[string]any{"flag_id": input.FlagID, "user_id": input.UserID, "revision": snapshot.Revision, "reason": evaluation.Reason})

		return &oc.Decision{Evaluation: evaluation, UserID: input.UserID, Revision: snapshot.Revision}, nil
	}

	return nil, oc.ErrNotFound
}
