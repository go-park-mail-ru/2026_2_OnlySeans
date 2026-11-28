package reviews_test

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type fixedReviewPolicy struct {
	policy domain.ReviewPolicy
}

func (p fixedReviewPolicy) ReviewPolicy(domain.UserID) domain.ReviewPolicy {
	return p.policy
}

func enabledPolicy() domain.ReviewPolicyProvider {
	return fixedReviewPolicy{policy: domain.ReviewPolicy{
		WriteEnabled: true,
		MaxLength:    domain.MaxReviewContentLen,
		Revision:     1,
	}}
}
