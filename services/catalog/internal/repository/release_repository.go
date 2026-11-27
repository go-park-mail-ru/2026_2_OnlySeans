package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type ReleaseRepository interface {
	List(context.Context, domain.ReleaseFilter, int64) ([]domain.Release, int64, error)
	Update(context.Context, int64, domain.ReleaseInput) error
	SetSubscription(context.Context, int64, int64, bool) error
	Notifications(context.Context, int64, int, int) ([]domain.Notification, int64, error)
	ReadNotification(context.Context, int64, int64) error
	DeliverReminders(context.Context, time.Time) (int64, error)
}
