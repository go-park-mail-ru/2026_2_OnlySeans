package onlineconf

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	DefaultListLimit = 50
	MaxListLimit     = 100
)

type ListOptions struct {
	Scope  string
	Query  string
	Limit  int
	Offset int
}

type AuditFilter struct {
	ResourceKind ResourceKind
	ResourceID   int64
	Limit        int
	Offset       int
}

type Repository interface {
	CreateScope(context.Context, domain.UserID, Scope) (*Scope, error)
	GetScope(context.Context, int64) (*Scope, error)
	ListScopes(context.Context, ListOptions) ([]Scope, int, error)
	UpdateScope(context.Context, domain.UserID, Scope) (*Scope, error)
	DeleteScope(context.Context, domain.UserID, int64, int64) error
	CreateConfig(context.Context, domain.UserID, Config) (*Config, error)
	GetConfig(context.Context, int64) (*Config, error)
	ListConfigs(context.Context, ListOptions) ([]Config, int, error)
	UpdateConfig(context.Context, domain.UserID, Config) (*Config, error)
	DeleteConfig(context.Context, domain.UserID, int64, int64) error
	CreateFlag(context.Context, domain.UserID, FeatureFlag) (*FeatureFlag, error)
	GetFlag(context.Context, int64) (*FeatureFlag, error)
	ListFlags(context.Context, ListOptions) ([]FeatureFlag, int, error)
	UpdateFlag(context.Context, domain.UserID, FeatureFlag) (*FeatureFlag, error)
	DeleteFlag(context.Context, domain.UserID, int64, int64) error
	ListAudit(context.Context, AuditFilter) ([]AuditEntry, int, error)
	LoadSnapshot(context.Context) (*Snapshot, error)
}
