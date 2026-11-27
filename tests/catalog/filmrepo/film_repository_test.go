package films_test

import "testing"

func TestFilmRepositoryGet(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetFilmByID",
			run:  checkPostgresService_GetFilmByID,
		},
		{
			name: "GetFilmByIDNullFields",
			run:  checkPostgresService_GetFilmByIDNullFields,
		},
		{
			name: "GetFilmByIDErrors",
			run:  checkPostgresService_GetFilmByIDErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFilmRepositoryList(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "ListFilms",
			run:  checkPostgresService_ListFilms,
		},
		{
			name: "ListFilmsEmptyPage",
			run:  checkPostgresService_ListFilmsEmptyPage,
		},
		{
			name: "ListFilmsErrors",
			run:  checkPostgresService_ListFilmsErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFilmRepositoryCollection(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetCollectionByID",
			run:  checkPostgresService_GetCollectionByID,
		},
		{
			name: "GetCollectionByIDErrors",
			run:  checkPostgresService_GetCollectionByIDErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFilmRepositoryCollections(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "ListCollections",
			run:  checkPostgresService_ListCollections,
		},
		{
			name: "ListCollectionsErrors",
			run:  checkPostgresService_ListCollectionsErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFilmRepositoryCollectionFilms(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "ListCollectionFilms",
			run:  checkPostgresService_ListCollectionFilms,
		},
		{
			name: "ListCollectionFilmsMissingCollection",
			run:  checkPostgresService_ListCollectionFilmsMissingCollection,
		},
		{
			name: "ListCollectionFilmsErrors",
			run:  checkPostgresService_ListCollectionFilmsErrors,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestFilmRepositoryCreate(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "CreateFilm",
			run:  checkPostgresService_CreateFilm,
		},
		{
			name: "CreateFilmError",
			run:  checkPostgresService_CreateFilmError,
		},
		{
			name: "CreateFilmOptionalFieldsAndSQLPayload",
			run:  checkPostgresServiceCreateFilmOptionalFieldsAndSQLPayload,
		},
		{
			name: "CreateFilmScanError",
			run:  checkPostgresServiceCreateFilmScanError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
