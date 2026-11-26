package postgres_test

import "testing"

func TestRatingRepositorySet(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "Set",
			run:  checkPostgresRatingRepo_Set,
		},
		{
			name: "SetMapsDatabaseErrors",
			run:  checkPostgresRatingRepo_SetMapsDatabaseErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestRatingRepositorySummaries(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetFilmRatingSummary",
			run:  checkPostgresRatingRepo_GetFilmRatingSummary,
		},
		{
			name: "GetSummaries",
			run:  checkPostgresRatingRepo_GetSummaries,
		},
		{
			name: "GetSummariesEmpty",
			run:  checkPostgresRatingRepo_GetSummariesEmpty,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestRatingRepositoryCollection(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetCollectionFilmRatingSummaries",
			run:  checkPostgresRatingRepo_GetCollectionFilmRatingSummaries,
		},
		{
			name: "GetCollectionFilmRatingSummariesEmptyPage",
			run:  checkPostgresRatingRepo_GetCollectionFilmRatingSummariesEmptyPage,
		},
		{
			name: "GetCollectionFilmRatingSummariesMissingCollection",
			run:  checkPostgresRatingRepo_GetCollectionFilmRatingSummariesMissingCollection,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
