package onlineconf

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	SnapshotAPIPath        = "/internal/onlineconf/snapshot"
	MaxSnapshotBytes       = 4 * 1024 * 1024
	snapshotJSONTerminator = "\n"
)

type SnapshotPayload struct {
	Revision    int64            `json:"revision"`
	GeneratedAt time.Time        `json:"generated_at"`
	Configs     []SnapshotConfig `json:"configs"`
	Flags       []SnapshotFlag   `json:"flags"`
}

type SnapshotConfig struct {
	Scope string          `json:"scope"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type SnapshotFlag struct {
	ID              int64           `json:"id"`
	Scope           string          `json:"scope"`
	Key             string          `json:"key"`
	Enabled         bool            `json:"enabled"`
	RolloutPercent  int             `json:"rollout_percent"`
	Salt            string          `json:"salt"`
	ForceOnUserIDs  []domain.UserID `json:"force_on_user_ids"`
	ForceOffUserIDs []domain.UserID `json:"force_off_user_ids"`
}

type snapshotResourceKey struct {
	scope string
	key   string
}

func (f SnapshotFlag) AsFeatureFlag() FeatureFlag {
	return FeatureFlag{ID: f.ID, Scope: f.Scope, Key: f.Key, Enabled: f.Enabled, RolloutPercent: f.RolloutPercent, Salt: f.Salt, ForceOnUserIDs: slices.Clone(f.ForceOnUserIDs), ForceOffUserIDs: slices.Clone(f.ForceOffUserIDs)}
}

func NewSnapshotPayload(snapshot *Snapshot) *SnapshotPayload {
	if snapshot == nil {
		return nil
	}

	payload := &SnapshotPayload{Revision: snapshot.Revision, GeneratedAt: snapshot.GeneratedAt.UTC(), Configs: make([]SnapshotConfig, 0, len(snapshot.Configs)), Flags: make([]SnapshotFlag, 0, len(snapshot.Flags))}
	for _, config := range snapshot.Configs {
		payload.Configs = append(payload.Configs, SnapshotConfig{Scope: config.Scope, Key: config.Key, Value: slices.Clone(config.Value)})
	}

	for _, flag := range snapshot.Flags {
		payload.Flags = append(payload.Flags, SnapshotFlag{ID: flag.ID, Scope: flag.Scope, Key: flag.Key, Enabled: flag.Enabled, RolloutPercent: flag.RolloutPercent, Salt: flag.Salt, ForceOnUserIDs: append([]domain.UserID{}, flag.ForceOnUserIDs...), ForceOffUserIDs: append([]domain.UserID{}, flag.ForceOffUserIDs...)})
	}

	return payload
}

func (s SnapshotPayload) Validate() error {
	if s.Revision < 0 || s.GeneratedAt.IsZero() || s.Configs == nil || s.Flags == nil {
		return ErrInvalidSnapshot
	}

	configs := make(map[snapshotResourceKey]struct{}, len(s.Configs))

	for _, config := range s.Configs {
		if err := ValidateKey(config.Scope); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		if err := ValidateKey(config.Key); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		if err := ValidateConfigValue(config.Scope, config.Key, config.Value); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		key := snapshotResourceKey{scope: config.Scope, key: config.Key}
		if _, exists := configs[key]; exists {
			return fmt.Errorf("%w: duplicate config", ErrInvalidSnapshot)
		}

		configs[key] = struct{}{}
	}

	flags := make(map[snapshotResourceKey]struct{}, len(s.Flags))
	ids := make(map[int64]struct{}, len(s.Flags))

	for _, flag := range s.Flags {
		if flag.ID < MinUserID || flag.ID > MaxSafeID {
			return fmt.Errorf("%w: invalid flag ID", ErrInvalidSnapshot)
		}

		if err := ValidateKey(flag.Scope); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		if err := ValidateKey(flag.Key); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		value := flag.AsFeatureFlag()
		if err := value.ValidateRules(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidSnapshot, err)
		}

		key := snapshotResourceKey{scope: flag.Scope, key: flag.Key}

		if _, exists := flags[key]; exists {
			return fmt.Errorf("%w: duplicate flag", ErrInvalidSnapshot)
		}

		if _, exists := ids[flag.ID]; exists {
			return fmt.Errorf("%w: duplicate flag ID", ErrInvalidSnapshot)
		}

		flags[key] = struct{}{}
		ids[flag.ID] = struct{}{}
	}

	if err := s.ValidateSize(); err != nil {
		return err
	}

	return nil
}

func (s SnapshotPayload) ValidateSize() error {
	encoded, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encode snapshot size: %w", err)
	}

	if len(encoded) > MaxSnapshotBytes-len(snapshotJSONTerminator) {
		return fmt.Errorf("%w: limit %d bytes", ErrSnapshotTooLarge, MaxSnapshotBytes)
	}

	return nil
}
