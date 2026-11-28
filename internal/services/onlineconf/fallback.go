package onlineconf

import (
	"encoding/json"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"
)

// FallbackProvider is used when OnlineConf is not configured or before the
// first successful snapshot: writes fail closed and the domain ceiling remains.
type FallbackProvider struct{}

func (FallbackProvider) ReviewPolicy(domain.UserID) domain.ReviewPolicy {
	return domain.ReviewPolicy{WriteEnabled: false, MaxLength: domain.MaxReviewContentLen}
}

func (FallbackProvider) PublicRuntime(userID domain.UserID) domain.PublicRuntime {
	result := (*Snapshot)(nil).PublicRuntime(userID)
	result.Flags[domain.LibraryAddKey] = true // Standalone Frame has no configured OnlineConf service.
	return result
}

func (FallbackProvider) GetConfig(_, _ string) (json.RawMessage, bool) {
	return nil, false
}

func (FallbackProvider) EvaluateFlag(_, _ string, userID domain.UserID) oc.Evaluation {
	return evaluator.Evaluate(nil, userID)
}
