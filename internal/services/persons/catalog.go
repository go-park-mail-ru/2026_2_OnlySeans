package persons

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type CatalogRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

func NewCatalogRepo(db *sql.DB, appLogger logger.Logger) *CatalogRepo {
	return &CatalogRepo{appLogger: appLogger, db: db}
}

const (
	catalogColumns = `
		p.id, btrim(concat_ws(' ', p.first_name, p.last_name)), COALESCE(p.original_name, ''),
		COALESCE(ph.external_url, '/static/' || ph.storage_key),
		COALESCE(ph.source_url, ''), COALESCE(ph.source_credit, ''),
		p.birth_date, p.death_date, COALESCE(p.birth_place, ''), COALESCE(p.biography, ''), COALESCE(p.source_url, '')`
	catalogFrom = ` FROM person AS p LEFT JOIN file AS ph ON ph.id = p.photo_file_id`

	filmPersonsQuery = `SELECT ` + catalogColumns + `, rt.code, COALESCE(fp.character_name, '')` + catalogFrom + `
		JOIN film_person AS fp ON fp.person_id = p.id
		JOIN role_type AS rt ON rt.id = fp.role_type_id
		WHERE fp.film_id = $1
		ORDER BY CASE rt.code WHEN 'director' THEN 0 WHEN 'actor' THEN 1 ELSE 2 END, fp.id`

	personQuery = `SELECT ` + catalogColumns + catalogFrom + ` WHERE p.id = $1`

	personFilmsQuery = `
		SELECT fp.film_id, rt.code, COALESCE(fp.character_name, '')
		FROM film_person AS fp
		JOIN role_type AS rt ON rt.id = fp.role_type_id
		WHERE fp.person_id = $1
		ORDER BY fp.film_id, fp.id`
)

type catalogScanner interface{ Scan(...any) error }

func scanCatalogPerson(row catalogScanner) (domain.CatalogPerson, error) {
	var p domain.CatalogPerson
	err := row.Scan(&p.ID, &p.Name, &p.OriginalName, &p.PhotoURL, &p.PhotoSource, &p.PhotoCredit, &p.BirthDate, &p.DeathDate, &p.BirthPlace, &p.Biography, &p.SourceURL)
	return p, err
}
func (r *CatalogRepo) FilmPersons(ctx context.Context, id int64) (_ []domain.PersonCredit, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person_catalog.film_persons", map[string]any{"film_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	persons := []domain.PersonCredit{}
	operationLog.Stage = "query"
	rows, err := r.db.QueryContext(ctx, filmPersonsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.PersonCredit
		p := &item.Person
		if err = rows.Scan(&p.ID, &p.Name, &p.OriginalName, &p.PhotoURL, &p.PhotoSource, &p.PhotoCredit, &p.BirthDate, &p.DeathDate, &p.BirthPlace, &p.Biography, &p.SourceURL, &item.Role, &item.Character); err != nil {
			return nil, err
		}
		persons = append(persons, item)
	}
	return persons, rows.Err()
}
func (r *CatalogRepo) Get(ctx context.Context, id int64) (_ domain.CatalogPerson, _ []domain.FilmCredit, operationErr error) {
	operationLog := storage.StartOperation(ctx, r.appLogger, "postgres", "person_catalog.get", map[string]any{"person_id": id})
	defer func() { operationLog.Finish(operationErr) }()

	operationLog.Stage = "query"
	person, err := scanCatalogPerson(r.db.QueryRowContext(ctx, personQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return person, nil, domain.ErrPersonNotFound
	}
	if err != nil {
		return person, nil, err
	}
	credits := []domain.FilmCredit{}
	operationLog.Stage = "query"
	rows, err := r.db.QueryContext(ctx, personFilmsQuery, id)
	if err != nil {
		return person, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var credit domain.FilmCredit
		if err = rows.Scan(&credit.FilmID, &credit.Role, &credit.Character); err != nil {
			return person, nil, err
		}
		credits = append(credits, credit)
	}
	return person, credits, rows.Err()
}
