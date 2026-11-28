package onlineconf

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const ProductBackendScope = "backend"

type resourceKey struct {
	scope string
	key   string
}

type flagRule struct {
	enabled  bool
	rollout  int
	included map[domain.UserID]struct{}
	excluded map[domain.UserID]struct{}
}

// Snapshot is immutable after construction and safe to publish atomically.
type Snapshot struct {
	revision int64
	configs  map[resourceKey]json.RawMessage
	flags    map[resourceKey]flagRule
}

func (s *Snapshot) Revision() int64 {
	if s == nil {
		return 0
	}
	return s.revision
}

func buildSnapshot(raw snapshotDTO) (*Snapshot, error) {
	if raw.Revision <= 0 {
		return nil, errors.New("snapshot revision must be positive")
	}
	snapshot := &Snapshot{
		revision: raw.Revision,
		configs:  make(map[resourceKey]json.RawMessage, len(raw.Configs)),
		flags:    make(map[resourceKey]flagRule, len(raw.Flags)),
	}
	for _, item := range raw.Configs {
		key, err := validatedKey(item.Scope, item.Key)
		if err != nil {
			return nil, err
		}
		if _, exists := snapshot.configs[key]; exists {
			return nil, fmt.Errorf("duplicate config %s/%s", key.scope, key.key)
		}
		if len(item.Value) == 0 || !json.Valid(item.Value) {
			return nil, fmt.Errorf("invalid config value %s/%s", key.scope, key.key)
		}
		value := append(json.RawMessage(nil), item.Value...)
		if key.scope == ProductBackendScope && key.key == domain.ReviewMaxLengthKey {
			var limit int
			if err := json.Unmarshal(value, &limit); err != nil || limit < domain.MinReviewContentLen || limit > domain.MaxReviewContentLen {
				return nil, fmt.Errorf("%s must be an integer from %d to %d", domain.ReviewMaxLengthKey, domain.MinReviewContentLen, domain.MaxReviewContentLen)
			}
		}
		snapshot.configs[key] = value
	}
	for _, item := range raw.Flags {
		key, err := validatedKey(item.Scope, item.Key)
		if err != nil {
			return nil, err
		}
		if _, exists := snapshot.flags[key]; exists {
			return nil, fmt.Errorf("duplicate flag %s/%s", key.scope, key.key)
		}
		rollout := 100
		if item.Rollout != nil {
			rollout = *item.Rollout
		}
		if rollout < 0 || rollout > 100 {
			return nil, fmt.Errorf("rollout for %s/%s must be from 0 to 100", key.scope, key.key)
		}
		rule := flagRule{enabled: item.Enabled, rollout: rollout, included: make(map[domain.UserID]struct{}), excluded: make(map[domain.UserID]struct{})}
		for _, id := range item.IncludedUserIDs {
			if id <= 0 {
				return nil, fmt.Errorf("included user id for %s/%s must be positive", key.scope, key.key)
			}
			rule.included[id] = struct{}{}
		}
		for _, id := range item.ExcludedUserIDs {
			if id <= 0 {
				return nil, fmt.Errorf("excluded user id for %s/%s must be positive", key.scope, key.key)
			}
			rule.excluded[id] = struct{}{}
		}
		snapshot.flags[key] = rule
	}
	return snapshot, nil
}

func validatedKey(scope, key string) (resourceKey, error) {
	scope = strings.TrimSpace(scope)
	key = strings.TrimSpace(key)
	if scope == "" || key == "" {
		return resourceKey{}, errors.New("snapshot scope and key must not be empty")
	}
	return resourceKey{scope: scope, key: key}, nil
}

func (s *Snapshot) reviewPolicy(userID domain.UserID) domain.ReviewPolicy {
	policy := domain.ReviewPolicy{MaxLength: domain.MaxReviewContentLen}
	if s == nil {
		return policy
	}
	policy.Revision = s.revision
	if raw, ok := s.configs[resourceKey{scope: ProductBackendScope, key: domain.ReviewMaxLengthKey}]; ok {
		_ = json.Unmarshal(raw, &policy.MaxLength)
	}
	if userID > 0 {
		if rule, ok := s.flags[resourceKey{scope: ProductBackendScope, key: domain.ReviewWriteEnabledKey}]; ok {
			policy.WriteEnabled = evaluate(rule, ProductBackendScope, domain.ReviewWriteEnabledKey, userID)
		}
	}
	return policy
}

func evaluate(rule flagRule, scope, key string, userID domain.UserID) bool {
	if _, excluded := rule.excluded[userID]; excluded {
		return false
	}
	if _, included := rule.included[userID]; included {
		return true
	}
	if !rule.enabled || rule.rollout <= 0 {
		return false
	}
	if rule.rollout >= 100 {
		return true
	}
	sum := sha256.Sum256([]byte(scope + "\x00" + key + "\x00" + strconv.Itoa(int(userID))))
	bucket := binary.BigEndian.Uint64(sum[:8]) % 100
	return bucket < uint64(rule.rollout)
}
