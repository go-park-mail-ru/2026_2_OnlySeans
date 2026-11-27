package usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/sessionpolicy"
)

func TestDailyRefreshUsesMoscowCalendarDay(t *testing.T) {
	now := time.Date(2026, time.November, 28, 0, 5, 0, 0, time.FixedZone("Moscow", sessionpolicy.MoscowOffset))
	require.True(t, sessionpolicy.NeedsRefresh(time.Time{}, now))
	require.True(t, sessionpolicy.NeedsRefresh(now.Add(-10*time.Minute), now))
	require.False(t, sessionpolicy.NeedsRefresh(now.Add(-time.Minute), now))
	require.False(t, sessionpolicy.NeedsRefresh(now.Add(time.Hour), now))
	require.False(t, sessionpolicy.NeedsRefresh(now.UTC(), now))
}
