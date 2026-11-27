package usecase

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

func (in SearchPersonsInput) Validate() error {
	if err := in.Criteria.Validate(); err != nil {
		return err
	}

	switch in.Sort {
	case "", PersonSearchSortRelevance, PersonSearchSortAddedAt:
	default:
		return fmt.Errorf("invalid person sort %q", in.Sort)
	}

	return searchquery.ValidatePage(in.Limit, in.Offset)
}

func (in CountPersonsInput) Validate() error {
	return in.Criteria.Validate()
}

func (criteria PersonSearchCriteria) Validate() error {
	if _, err := searchquery.NormalizeQuery(criteria.Query); err != nil {
		return err
	}

	if err := criteria.Roles.Validate(2, validSearchPersonRole); err != nil {
		return fmt.Errorf("roles: %w", err)
	}

	return nil
}

func validSearchPersonRole(role SearchPersonRole) bool {
	return role == SearchPersonRoleActor || role == SearchPersonRoleDirector
}
