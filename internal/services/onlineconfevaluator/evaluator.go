package onlineconfevaluator

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	hashInputSeparator = ":"
	userIDBase         = 10
	bucketDigestBytes  = 8
	bucketCount        = 10000
	bucketsPerPercent  = bucketCount / oc.MaxRolloutPercent
)

func Evaluate(flag *oc.FeatureFlag, userID domain.UserID) oc.Evaluation {
	if flag == nil || userID < oc.MinUserID || int64(userID) > oc.MaxSafeID || !flag.Enabled {
		return oc.Evaluation{Reason: oc.ReasonGlobalDisabled}
	}

	for _, id := range flag.ForceOffUserIDs {
		if id == userID {
			return oc.Evaluation{Reason: oc.ReasonForceOff}
		}
	}

	for _, id := range flag.ForceOnUserIDs {
		if id == userID {
			return oc.Evaluation{Enabled: true, Reason: oc.ReasonForceOn}
		}
	}

	digest := sha256.Sum256([]byte(flag.Salt + hashInputSeparator + strconv.FormatInt(int64(userID), userIDBase)))
	bucket := int(binary.BigEndian.Uint64(digest[:bucketDigestBytes]) % bucketCount)
	enabled := flag.RolloutPercent >= oc.MinRolloutPercent && flag.RolloutPercent <= oc.MaxRolloutPercent &&
		bucket < flag.RolloutPercent*bucketsPerPercent
	reason := oc.ReasonRolloutExcluded

	if enabled {
		reason = oc.ReasonRolloutIncluded
	}

	return oc.Evaluation{Enabled: enabled, Reason: reason, Bucket: &bucket}
}
