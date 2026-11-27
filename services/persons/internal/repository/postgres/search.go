package persons

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

type PersonSearchRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

var _ usecase.PersonSearchRepository = (*PersonSearchRepo)(nil)

func NewPersonSearchRepo(db *sql.DB, appLogger logger.Logger) *PersonSearchRepo {
	return &PersonSearchRepo{db: db, appLogger: appLogger}
}

func (r *PersonSearchRepo) SearchPersons(ctx context.Context, input usecase.SearchPersonsInput) (_ usecase.SearchPersonsOutput, operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, "postgres", "person_catalog.search_filtered", nil)
	defer func() { operation.Finish(operationErr) }()

	args := append(personSearchArgs(input.Criteria), input.Limit, input.Offset, input.IncludeWorksCount)
	query := personSearchMatches + fmt.Sprintf(personSearchPage, personSearchOrder(input.Sort))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return usecase.SearchPersonsOutput{}, fmt.Errorf("query person search page: %w", err)
	}
	defer func() { _ = rows.Close() }()

	output := usecase.SearchPersonsOutput{Persons: make([]usecase.PersonSuggestion, 0)}

	for rows.Next() {
		var (
			suggestion usecase.PersonSuggestion
			id         *int64
			roles      []byte
		)

		person := &suggestion.Person

		err := rows.Scan(&output.Total, &id, &person.Name, &person.OriginalName, &person.PhotoURL, &roles, &suggestion.WorksCount)
		if err != nil {
			return usecase.SearchPersonsOutput{}, fmt.Errorf("scan person search result: %w", err)
		}

		// An empty page carries only the total and has no person ID.
		if id == nil {
			continue
		}

		person.ID = *id

		if err := json.Unmarshal(roles, &suggestion.Roles); err != nil {
			return usecase.SearchPersonsOutput{}, fmt.Errorf("decode person search roles: %w", err)
		}

		output.Persons = append(output.Persons, suggestion)
	}

	if err := rows.Err(); err != nil {
		return usecase.SearchPersonsOutput{}, fmt.Errorf("read person search page: %w", err)
	}

	return output, nil
}

func (r *PersonSearchRepo) CountPersons(ctx context.Context, criteria usecase.PersonSearchCriteria) (_ int64, operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, "postgres", "person_catalog.count_matches", nil)
	defer func() { operation.Finish(operationErr) }()

	var total int64

	err := r.db.QueryRowContext(ctx, personSearchMatches+`SELECT total FROM totals`, personSearchArgs(criteria)...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count person matches: %w", err)
	}

	return total, nil
}

func personSearchArgs(criteria usecase.PersonSearchCriteria) []any {
	prefix, contains := searchquery.Patterns(criteria.Query)

	return []any{criteria.Query, prefix, contains, criteria.Roles.Include, criteria.Roles.Exclude,
		criteria.Roles.Mode == searchquery.FilterAny}
}

func personSearchOrder(sort usecase.PersonSearchSort) string {
	if sort == usecase.PersonSearchSortAddedAt {
		return "m.created_at DESC"
	}

	return "m.match_kind, m.score DESC"
}
