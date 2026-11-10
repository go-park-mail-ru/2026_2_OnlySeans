package persons

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	personColumns = `
		p.id, p.first_name, p.last_name, p.birth_date,
		(SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
			FROM file AS fl WHERE fl.id = p.photo_file_id),
		p.created_at, p.updated_at`

	getPersonQuery = `
		SELECT ` + personColumns + `
		FROM person AS p
		WHERE p.id = $1`

	listPersonsQuery = `
		SELECT ` + personColumns + `
		FROM person AS p
		ORDER BY p.id
		LIMIT $1 OFFSET $2`

	countPersonsQuery = `
		SELECT count(*) FROM person`

	createPersonQuery = `
		INSERT INTO person (first_name, last_name, birth_date)
		VALUES ($1, $2, $3)
		RETURNING id`

	updatePersonQuery = `
		UPDATE person AS p
		SET first_name = $2, last_name = $3, birth_date = $4
		WHERE p.id = $1
		RETURNING ` + personColumns

	deletePersonQuery = `
		DELETE FROM person
		WHERE id = $1`
)

type PersonRepo struct {
	appLogger logger.Logger
	db        *sql.DB
}

func NewPersonRepo(db *sql.DB, appLogger logger.Logger) domain.PersonRepository {
	return &PersonRepo{appLogger: appLogger,
		db: db,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPerson(row rowScanner) (domain.Person, error) {
	var person domain.Person

	err := row.Scan(
		&person.ID,
		&person.FirstName,
		&person.LastName,
		&person.BirthDate,
		&person.PhotoID,
		&person.CreatedAt,
		&person.UpdatedAt,
	)

	return person, err
}

func (r *PersonRepo) GetByID(
	ctx context.Context,
	id domain.PersonID,
) (_ *domain.Person, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person.get_by_id", map[string]any{"person_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	person, err := scanPerson(
		r.db.QueryRowContext(ctx, getPersonQuery, id),
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPersonNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get person %d: %w", id, err)
	}

	return &person, nil
}

func (s *PersonRepo) List(ctx context.Context, limit, offset int) (_ []domain.Person, _ int, operationErr error) {
	operationLog := storage.StartOperation(ctx, s.appLogger, "postgres", "person.list", map[string]any{"limit": limit, "offset": offset})
	defer func() { operationLog.Finish(operationErr) }()

	var total int

	operationLog.Stage = "query"
	if err := s.db.QueryRowContext(ctx, countPersonsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count persons: %w", err)
	}

	operationLog.Stage = "query"
	rows, err := s.db.QueryContext(ctx, listPersonsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list persons: %w", err)
	}
	defer rows.Close()

	list := make([]domain.Person, 0)

	for rows.Next() {
		person, err := scanPerson(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan person: %w", err)
		}

		list = append(list, person)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list persons: %w", err)
	}

	return list, total, nil
}

func (r *PersonRepo) Create(
	ctx context.Context,
	input domain.PersonInput,
) (_ domain.PersonID, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person.create", map[string]any{})
	defer func() { operationLog.Finish(operationErr) }()

	var id domain.PersonID

	operationLog.Stage = "query"
	err := r.db.QueryRowContext(
		ctx,
		createPersonQuery,
		input.FirstName,
		input.LastName,
		input.BirthDate,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create person: %w", err)
	}

	return id, nil
}

func (r *PersonRepo) Update(
	ctx context.Context,
	id domain.PersonID,
	input domain.PersonInput,
) (_ *domain.Person, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person.update", map[string]any{"person_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	person, err := scanPerson(
		r.db.QueryRowContext(
			ctx,
			updatePersonQuery,
			id,
			input.FirstName,
			input.LastName,
			input.BirthDate,
		),
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPersonNotFound
	}

	if err != nil {

		return nil, fmt.Errorf("update person %d: %w", id, err)
	}

	return &person, nil
}

func (r *PersonRepo) Delete(
	ctx context.Context,
	id domain.PersonID,
) (operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person.delete", map[string]any{"person_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	result, err := r.db.ExecContext(ctx, deletePersonQuery, id)
	if err != nil {

		return fmt.Errorf("delete person %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {

		return fmt.Errorf("delete person %d: %w", id, err)
	}

	if affected == 0 {
		return domain.ErrPersonNotFound
	}

	return nil
}
