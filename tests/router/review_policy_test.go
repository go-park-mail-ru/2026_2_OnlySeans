package router_test

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type routerReviewPolicy struct{}

func (routerReviewPolicy) ReviewPolicy(domain.UserID) domain.ReviewPolicy {
	return domain.ReviewPolicy{WriteEnabled: true, MaxLength: domain.MaxReviewContentLen, Revision: 1}
}
