package onlineconf

import (
	"encoding/json"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type AdminIdentity struct {
	ID       domain.UserID `json:"id"`
	Username string        `json:"username"`
	Role     domain.Role   `json:"role"`
}

type Scope struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Config struct {
	ID          int64           `json:"id"`
	Scope       string          `json:"scope"`
	Key         string          `json:"key"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Value       json.RawMessage `json:"value"`
	Version     int64           `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Override struct {
	UserID  domain.UserID `json:"user_id"`
	Enabled bool          `json:"enabled"`
}

type FeatureFlag struct {
	ID              int64           `json:"id"`
	Scope           string          `json:"scope"`
	Key             string          `json:"key"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Enabled         bool            `json:"enabled"`
	RolloutPercent  int             `json:"rollout_percent"`
	Salt            string          `json:"salt"`
	ForceOnUserIDs  []domain.UserID `json:"force_on_user_ids"`
	ForceOffUserIDs []domain.UserID `json:"force_off_user_ids"`
	Version         int64           `json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionUpdate AuditAction = "update"
	ActionDelete AuditAction = "delete"
)

type ResourceKind string

const (
	ResourceScope  ResourceKind = "scope"
	ResourceConfig ResourceKind = "config"
	ResourceFlag   ResourceKind = "flag"
)

type AuditEntry struct {
	ID           int64           `json:"id"`
	ActorID      domain.UserID   `json:"actor_id"`
	ActorName    string          `json:"actor_name"`
	Action       AuditAction     `json:"action"`
	ResourceKind ResourceKind    `json:"resource_type"`
	Scope        string          `json:"scope"`
	Key          string          `json:"key"`
	ResourceID   int64           `json:"resource_id"`
	Before       json.RawMessage `json:"before"`
	After        json.RawMessage `json:"after"`
	Revision     int64           `json:"revision"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Snapshot struct {
	Revision    int64         `json:"revision"`
	GeneratedAt time.Time     `json:"generated_at"`
	Scopes      []Scope       `json:"scopes"`
	Configs     []Config      `json:"configs"`
	Flags       []FeatureFlag `json:"flags"`
}

type DecisionReason string

const (
	ReasonGlobalDisabled  DecisionReason = "globally_disabled"
	ReasonForceOff        DecisionReason = "forced_off"
	ReasonForceOn         DecisionReason = "forced_on"
	ReasonRolloutIncluded DecisionReason = "rollout_match"
	ReasonRolloutExcluded DecisionReason = "rollout_miss"
)

type Evaluation struct {
	Enabled bool           `json:"enabled"`
	Reason  DecisionReason `json:"reason"`
	Bucket  *int           `json:"bucket"`
}

type Decision struct {
	Evaluation
	UserID   domain.UserID `json:"user_id"`
	Revision int64         `json:"revision"`
}

const (
	MinUserID                = 1
	MaxSafeID          int64 = 1<<53 - 1
	MinRolloutPercent        = 0
	MaxRolloutPercent        = 100
	ReviewsScope             = "reviews"
	ReviewMaxLengthKey       = "max_length"
	MinReviewLength          = 50
	MaxReviewLength          = 20000
)
