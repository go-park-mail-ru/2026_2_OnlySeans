package domain

import "errors"

var (
	ErrNilAnalyticsRepo       = errors.New("analytics repository is nil")
	ErrInvalidAnalyticsPeriod = errors.New("invalid analytics period")
	ErrInvalidAnalyticsLimit  = errors.New("invalid analytics limit")
)
