package sessionpolicy

import "time"

const (
	MoscowOffset = 3 * 60 * 60
	dayDuration  = 24 * time.Hour
)

func NeedsRefresh(lastRefreshedAt, now time.Time) bool {
	if lastRefreshedAt.IsZero() {
		return true
	}

	offset := time.Duration(MoscowOffset) * time.Second

	return lastRefreshedAt.UTC().Add(offset).Truncate(dayDuration).Before(now.UTC().Add(offset).Truncate(dayDuration))
}
