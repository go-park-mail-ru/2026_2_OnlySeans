package onlineconf_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

type useCaseRepository struct {
	oc.Repository
	createConfig func(context.Context, domain.UserID, oc.Config) (*oc.Config, error)
	updateConfig func(context.Context, domain.UserID, oc.Config) (*oc.Config, error)
	createFlag   func(context.Context, domain.UserID, oc.FeatureFlag) (*oc.FeatureFlag, error)
}

func (r *useCaseRepository) CreateConfig(ctx context.Context, actor domain.UserID, value oc.Config) (*oc.Config, error) {
	return r.createConfig(ctx, actor, value)
}

func (r *useCaseRepository) UpdateConfig(ctx context.Context, actor domain.UserID, value oc.Config) (*oc.Config, error) {
	return r.updateConfig(ctx, actor, value)
}

func (r *useCaseRepository) CreateFlag(ctx context.Context, actor domain.UserID, value oc.FeatureFlag) (*oc.FeatureFlag, error) {
	return r.createFlag(ctx, actor, value)
}

type refreshSpy struct{ calls int }

func (r *refreshSpy) RequestRefresh() { r.calls++ }

type snapshotStub struct {
	value *oc.Snapshot
	err   error
	calls int
}

func (s *snapshotStub) Current() (*oc.Snapshot, error) {
	s.calls++
	return s.value, s.err
}

type userReaderFunc func(context.Context, domain.UserID) (*domain.User, error)

func (f userReaderFunc) GetUser(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return f(ctx, id)
}

func TestConfigUseCaseRefreshAfterCommit(t *testing.T) {
	refresher := &refreshSpy{}
	repo := &useCaseRepository{createConfig: func(_ context.Context, actor domain.UserID, value oc.Config) (*oc.Config, error) {
		if actor != testActor || refresher.calls != 0 || value.Title != "Лимит" {
			t.Fatal("actor, normalization or premature refresh")
		}
		value.ID = 1
		value.Version = 1
		return &value, nil
	}}

	uc, err := usecase.NewConfigsUseCase(repo, refresher, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	result, err := uc.Create(context.Background(), usecase.CreateConfigInput{ActorID: testActor, Fields: usecase.ConfigFields{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: " Лимит ", Value: json.RawMessage(`500`)}})
	if err != nil || result.ID != 1 || refresher.calls != 1 {
		t.Fatalf("create: %v %v refresh=%d", result, err, refresher.calls)
	}
}

func TestConfigUseCaseConflictDoesNotRefresh(t *testing.T) {
	refresher := &refreshSpy{}
	repo := &useCaseRepository{updateConfig: func(_ context.Context, _ domain.UserID, value oc.Config) (*oc.Config, error) {
		if value.ID != 1 || value.Version != 3 {
			t.Fatal("ID/version lost")
		}
		return nil, oc.ErrVersionConflict
	}}

	uc, err := usecase.NewConfigsUseCase(repo, refresher, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	result, err := uc.Update(context.Background(), usecase.UpdateConfigInput{ActorID: testActor, ID: 1, Version: 3, Fields: usecase.ConfigFields{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: "Лимит", Value: json.RawMessage(`500`)}})
	if !errors.Is(err, oc.ErrVersionConflict) || result != nil || refresher.calls != 0 {
		t.Fatalf("conflict: %v %v refresh=%d", result, err, refresher.calls)
	}
}

func TestConfigUseCaseRejectsInvalidInputBeforeStorage(t *testing.T) {
	calls := 0
	refresher := &refreshSpy{}
	repo := &useCaseRepository{createConfig: func(context.Context, domain.UserID, oc.Config) (*oc.Config, error) { calls++; return nil, nil }}

	uc, err := usecase.NewConfigsUseCase(repo, refresher, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`49`), json.RawMessage(`20001`), json.RawMessage(`"500"`)} {
		if _, err := uc.Create(context.Background(), usecase.CreateConfigInput{ActorID: testActor, Fields: usecase.ConfigFields{Scope: oc.ReviewsScope, Key: oc.ReviewMaxLengthKey, Title: "Лимит", Value: value}}); err == nil {
			t.Fatalf("accepted invalid value: %s", value)
		}
	}

	if calls != 0 || refresher.calls != 0 {
		t.Fatal("invalid input reached storage or refreshed snapshot")
	}
}

func TestFlagsUseCaseRejectsConflictingOverrides(t *testing.T) {
	calls := 0
	refresher := &refreshSpy{}
	repo := &useCaseRepository{createFlag: func(context.Context, domain.UserID, oc.FeatureFlag) (*oc.FeatureFlag, error) {
		calls++
		return nil, nil
	}}

	uc, err := usecase.NewFlagsUseCase(repo, refresher, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Create(context.Background(), usecase.CreateFlagInput{ActorID: testActor, Fields: usecase.FlagFields{Scope: oc.ReviewsScope, Key: "write_enabled", Title: "Отзывы", Enabled: true, RolloutPercent: 25, ForceOnUserIDs: []domain.UserID{testActor}, ForceOffUserIDs: []domain.UserID{testActor}}})
	if !errors.Is(err, oc.ErrConflictingOverrides) || calls != 0 || refresher.calls != 0 {
		t.Fatalf("intersection: %v", err)
	}
}

func TestEvaluateUseCaseUsesSingleSnapshot(t *testing.T) {
	snapshots := &snapshotStub{value: &oc.Snapshot{Revision: 42, Flags: []oc.FeatureFlag{{ID: 7, Enabled: true, Salt: "a1b2", RolloutPercent: 0, ForceOnUserIDs: []domain.UserID{testActor}}}}}
	users := userReaderFunc(func(_ context.Context, id domain.UserID) (*domain.User, error) { return &domain.User{ID: id}, nil })

	uc, err := usecase.NewEvaluateFlagUseCase(users, snapshots, service.Evaluate, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	result, err := uc.Run(context.Background(), usecase.EvaluateFlagInput{FlagID: 7, UserID: testActor})
	if err != nil || !result.Enabled || result.Reason != oc.ReasonForceOn || result.Revision != 42 || result.UserID != testActor || result.Bucket != nil || snapshots.calls != 1 {
		t.Fatalf("evaluate: %v %v calls=%d", result, err, snapshots.calls)
	}
}

func TestEvaluateUseCaseMissingUserAndFlag(t *testing.T) {
	snapshots := &snapshotStub{value: &oc.Snapshot{Revision: 42}}
	users := userReaderFunc(func(context.Context, domain.UserID) (*domain.User, error) { return nil, domain.ErrUserNotFound })

	uc, err := usecase.NewEvaluateFlagUseCase(users, snapshots, service.Evaluate, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uc.Run(context.Background(), usecase.EvaluateFlagInput{FlagID: 7, UserID: testActor}); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}

	if snapshots.calls != 0 {
		t.Fatal("read snapshot for missing user")
	}

	users = userReaderFunc(func(_ context.Context, id domain.UserID) (*domain.User, error) { return &domain.User{ID: id}, nil })

	uc, err = usecase.NewEvaluateFlagUseCase(users, snapshots, service.Evaluate, testutil.DiscardLogger())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := uc.Run(context.Background(), usecase.EvaluateFlagInput{FlagID: 7, UserID: testActor}); !errors.Is(err, oc.ErrNotFound) {
		t.Fatalf("missing flag: %v", err)
	}

	snapshots.value = nil

	if _, err := uc.Run(context.Background(), usecase.EvaluateFlagInput{FlagID: 7, UserID: testActor}); !errors.Is(err, oc.ErrSnapshotNotReady) {
		t.Fatalf("missing snapshot: %v", err)
	}
}
