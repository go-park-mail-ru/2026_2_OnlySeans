package onlineconf_test

import (
	"testing"

	evaluator "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconfevaluator"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

func TestEvaluatorRulePriority(t *testing.T) {
	cases := []struct {
		name    string
		flag    *oc.FeatureFlag
		user    domain.UserID
		enabled bool
		reason  oc.DecisionReason
		bucket  bool
	}{
		{name: "missing", user: 1, reason: oc.ReasonGlobalDisabled},
		{name: "guest", flag: &oc.FeatureFlag{Enabled: true, RolloutPercent: 100}, reason: oc.ReasonGlobalDisabled},
		{name: "global disable beats force on", flag: &oc.FeatureFlag{ForceOnUserIDs: []domain.UserID{1}, RolloutPercent: 100}, user: 1, reason: oc.ReasonGlobalDisabled},
		{name: "force off beats force on on corrupt data", flag: &oc.FeatureFlag{Enabled: true, ForceOnUserIDs: []domain.UserID{1}, ForceOffUserIDs: []domain.UserID{1}, RolloutPercent: 100}, user: 1, reason: oc.ReasonForceOff},
		{name: "force on beats zero rollout", flag: &oc.FeatureFlag{Enabled: true, ForceOnUserIDs: []domain.UserID{1}}, user: 1, enabled: true, reason: oc.ReasonForceOn},
		{name: "force off beats full rollout", flag: &oc.FeatureFlag{Enabled: true, RolloutPercent: 100, ForceOffUserIDs: []domain.UserID{1}}, user: 1, reason: oc.ReasonForceOff},
		{name: "zero rollout", flag: &oc.FeatureFlag{Enabled: true, Salt: "a1b2"}, user: 1, reason: oc.ReasonRolloutExcluded, bucket: true},
		{name: "full rollout", flag: &oc.FeatureFlag{Enabled: true, Salt: "a1b2", RolloutPercent: 100}, user: 1, enabled: true, reason: oc.ReasonRolloutIncluded, bucket: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluator.Evaluate(tc.flag, tc.user)
			if got.Enabled != tc.enabled || got.Reason != tc.reason || (got.Bucket != nil) != tc.bucket {
				t.Fatalf("decision=%+v", got)
			}
		})
	}
}

func TestEvaluatorStableAndMonotonic(t *testing.T) {
	flag := oc.FeatureFlag{Enabled: true, Salt: "a1b2"}

	for id := domain.UserID(1); id <= 1000; id++ {
		included := false
		previousBucket := -1

		for _, percent := range []int{0, 1, 10, 25, 50, 99, 100} {
			flag.RolloutPercent = percent
			got := evaluator.Evaluate(&flag, id)
			again := evaluator.Evaluate(&flag, id)

			if got.Bucket == nil || *got.Bucket < 0 || *got.Bucket >= 10000 || *again.Bucket != *got.Bucket || again.Enabled != got.Enabled {
				t.Fatal("unstable decision")
			}

			if previousBucket >= 0 && *got.Bucket != previousBucket {
				t.Fatal("rollout changed bucket")
			}

			if included && !got.Enabled {
				t.Fatal("user left rollout as percentage increased")
			}

			if percent == 0 && got.Enabled || percent == 100 && !got.Enabled {
				t.Fatal("rollout boundary")
			}

			included = got.Enabled
			previousBucket = *got.Bucket
		}
	}
}

func TestEvaluatorContractVectorsAndStableSalt(t *testing.T) {
	flag := oc.FeatureFlag{Scope: "reviews", Key: "write_enabled", Salt: "a1b2", Enabled: true, RolloutPercent: 25}
	for _, vector := range []struct {
		user    domain.UserID
		bucket  int
		enabled bool
	}{{1, 4554, false}, {2, 6640, false}, {101, 1301, true}} {
		result := evaluator.Evaluate(&flag, vector.user)
		if result.Bucket == nil || *result.Bucket != vector.bucket || result.Enabled != vector.enabled {
			t.Fatalf("user=%d decision=%+v", vector.user, result)
		}

		flag.Title = "Updated title"
		flag.Version++

		after := evaluator.Evaluate(&flag, vector.user)
		if *after.Bucket != vector.bucket || after.Enabled != vector.enabled {
			t.Fatal("metadata changed rollout")
		}
	}
}
