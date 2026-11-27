package usecase

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

type SearchPersonRole string

const (
	SearchPersonRoleActor    SearchPersonRole = "actor"
	SearchPersonRoleDirector SearchPersonRole = "director"
)

type PersonSearchSort string

const (
	PersonSearchSortRelevance PersonSearchSort = "relevance"
	PersonSearchSortAddedAt   PersonSearchSort = "added_at"
)

type PersonSearchCriteria struct {
	Query string
	Roles searchquery.Filter[SearchPersonRole]
}

type SearchPersonsInput struct {
	Criteria          PersonSearchCriteria
	Sort              PersonSearchSort
	Limit             int
	Offset            int64
	IncludeWorksCount bool
}

type SearchPersonsOutput struct {
	Persons []PersonSuggestion
	Total   int64
}

type CountPersonsInput struct {
	Criteria PersonSearchCriteria
}

type CountPersonsOutput struct {
	Total int64
}

type PersonSuggestion struct {
	Person     domain.CatalogPerson
	Roles      []SearchPersonRole
	WorksCount *int64
}
