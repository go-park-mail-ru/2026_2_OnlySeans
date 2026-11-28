package domain

const (
	ReviewWriteEnabledKey = "reviews.write_enabled"
	ReviewMaxLengthKey    = "reviews.max_length"
)

// ReviewPolicy is one consistent decision calculated from a single config snapshot.
type ReviewPolicy struct {
	SnapshotReady bool
	WriteEnabled  bool
	MaxLength     int
	Revision      int64
}

// ReviewPolicyProvider is intentionally synchronous: implementations must read
// already cached state and must not perform HTTP or SQL on a product request.
type ReviewPolicyProvider interface {
	ReviewPolicy(userID UserID) ReviewPolicy
}
