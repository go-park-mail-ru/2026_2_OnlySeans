package usecase

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	releaseReminderInterval = time.Minute
	releaseReminderTimeout  = 15 * time.Second
)

func (u *ReleaseUseCase) RunReminders(ctx context.Context, appLogger logger.Logger) {
	tick := time.NewTicker(releaseReminderInterval)
	defer tick.Stop()

	for {
		batchCtx, cancel := context.WithTimeout(ctx, releaseReminderTimeout)
		count, err := u.repo.DeliverReminders(batchCtx, u.now())

		cancel()

		if err != nil && ctx.Err() == nil && appLogger != nil {
			appLogger.ErrorContext(ctx, "release reminders failed", map[string]any{"error": err.Error()})
		}

		if count > 0 && appLogger != nil {
			appLogger.InfoContext(ctx, "release reminders delivered", map[string]any{"count": count})
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}
