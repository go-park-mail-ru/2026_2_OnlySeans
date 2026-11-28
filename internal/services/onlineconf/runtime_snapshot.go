package onlineconf

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"
)

const (
	reviewsScope                 = oc.ReviewsScope
	frontendScope                = "frontend"
	maxLengthSnapshotKey         = oc.ReviewMaxLengthKey
	writeEnabledSnapshotKey      = "write_enabled"
	homeBlocksSnapshotKey        = "home_blocks"
	homeBlocksEnabledSnapshotKey = "home_blocks_enabled"
	emptyBlocksJSON              = "[]"
	publicKeySeparator           = "."
)

type resourceKey struct {
	scope string
	key   string
}

type Snapshot struct {
	revision int64
	configs  map[resourceKey]json.RawMessage
	flags    map[resourceKey]oc.FeatureFlag
}

func (s *Snapshot) Revision() int64 {
	if s == nil {
		return 0
	}

	return s.revision
}

func buildSnapshot(raw oc.SnapshotPayload) (*Snapshot, error) {
	if err := raw.Validate(); err != nil {
		return nil, fmt.Errorf("validate onlineconf snapshot: %w", err)
	}

	snapshot := &Snapshot{
		revision: raw.Revision,
		configs:  make(map[resourceKey]json.RawMessage, len(raw.Configs)),
		flags:    make(map[resourceKey]oc.FeatureFlag, len(raw.Flags)),
	}

	for _, item := range raw.Configs {
		snapshot.configs[resourceKey{scope: item.Scope, key: item.Key}] = bytes.Clone(item.Value)
	}

	for _, item := range raw.Flags {
		snapshot.flags[resourceKey{scope: item.Scope, key: item.Key}] = item.AsFeatureFlag()
	}

	return snapshot, nil
}

func (s *Snapshot) GetConfig(scope, key string) (json.RawMessage, bool) {
	if s == nil {
		return nil, false
	}

	value, ok := s.configs[resourceKey{scope: scope, key: key}]

	return bytes.Clone(value), ok
}

func (s *Snapshot) EvaluateFlag(scope, key string, userID domain.UserID) oc.Evaluation {
	if s == nil {
		return evaluator.Evaluate(nil, userID)
	}

	flag, ok := s.flags[resourceKey{scope: scope, key: key}]
	if !ok {
		return evaluator.Evaluate(nil, userID)
	}

	return evaluator.Evaluate(&flag, userID)
}

func (s *Snapshot) ReviewPolicy(userID domain.UserID) domain.ReviewPolicy {
	policy := domain.ReviewPolicy{MaxLength: domain.MaxReviewContentLen}
	if s == nil {
		return policy
	}

	policy.SnapshotReady = true
	policy.Revision = s.revision

	if raw, ok := s.configs[resourceKey{scope: reviewsScope, key: maxLengthSnapshotKey}]; ok {
		if err := json.Unmarshal(raw, &policy.MaxLength); err != nil {
			policy.MaxLength = domain.MaxReviewContentLen
		}
	}

	policy.WriteEnabled = s.EvaluateFlag(reviewsScope, writeEnabledSnapshotKey, userID).Enabled

	return policy
}

func (s *Snapshot) PublicRuntime(userID domain.UserID) domain.PublicRuntime {
	result := domain.PublicRuntime{
		ReviewPolicy: s.ReviewPolicy(userID),
		HomeBlocks:   json.RawMessage(emptyBlocksJSON),
		Configs:      make(map[string]json.RawMessage),
		Flags:        make(map[string]bool),
	}
	if s == nil {
		return result
	}

	for key, value := range s.configs {
		if publicScope(key.scope) {
			result.Configs[key.scope+publicKeySeparator+key.key] = bytes.Clone(value)
		}
	}

	for key := range s.flags {
		if publicScope(key.scope) {
			result.Flags[key.scope+publicKeySeparator+key.key] = s.EvaluateFlag(key.scope, key.key, userID).Enabled
		}
	}

	result.HomeBlocksEnabled = s.EvaluateFlag(frontendScope, homeBlocksEnabledSnapshotKey, userID).Enabled
	if result.HomeBlocksEnabled {
		blocks := bytes.TrimSpace(s.configs[resourceKey{scope: frontendScope, key: homeBlocksSnapshotKey}])
		if len(blocks) > 0 && blocks[0] == '[' {
			result.HomeBlocks = bytes.Clone(blocks)
		}
	}

	result.Configs[domain.HomeBlocksKey] = bytes.Clone(result.HomeBlocks)

	return result
}

func publicScope(scope string) bool {
	return scope == frontendScope
}
