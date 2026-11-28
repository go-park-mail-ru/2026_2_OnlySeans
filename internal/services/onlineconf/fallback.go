package onlineconf

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

// FallbackProvider is used when OnlineConf is not configured or before the
// first successful snapshot: writes fail closed and the domain ceiling remains.
type FallbackProvider struct{}

func (FallbackProvider) ReviewPolicy(domain.UserID) domain.ReviewPolicy {
	return domain.ReviewPolicy{WriteEnabled: false, MaxLength: domain.MaxReviewContentLen}
}

func (FallbackProvider) PublicRuntime(userID domain.UserID) domain.PublicRuntime {
	return (*Snapshot)(nil).publicRuntime(userID)
}
