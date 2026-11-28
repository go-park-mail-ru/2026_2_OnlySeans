package domain

import "encoding/json"

const (
	ReviewWriteEnabledKey = "reviews.write_enabled"
	ReviewMaxLengthKey    = "reviews.max_length"
	HomeBlocksKey         = "frontend.home_blocks"
	HomeBlocksEnabledKey  = "frontend.home_blocks_enabled"
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

// PublicRuntime carries only the settings exposed to the current browser session.
type PublicRuntime struct {
	ReviewPolicy
	HomeBlocks        json.RawMessage
	HomeBlocksEnabled bool
	Configs           map[string]json.RawMessage
	Flags             map[string]bool
}

type RuntimeProvider interface {
	ReviewPolicyProvider
	PublicRuntime(userID UserID) PublicRuntime
}
