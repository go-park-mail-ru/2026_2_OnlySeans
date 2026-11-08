package persons

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CatalogPerson struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	OriginalName string     `json:"original_name"`
	PhotoURL     *string    `json:"photo_url,omitempty"`
	PhotoSource  string     `json:"photo_source"`
	PhotoCredit  string     `json:"photo_credit"`
	BirthDate    *time.Time `json:"birth_date,omitempty"`
	DeathDate    *time.Time `json:"death_date,omitempty"`
	BirthPlace   string     `json:"birth_place"`
	Biography    string     `json:"biography"`
	SourceURL    string     `json:"source_url"`
}
type PersonCredit struct {
	Person    CatalogPerson `json:"person"`
	Role      string        `json:"role"`
	Character string        `json:"character"`
}
type FilmCredit struct {
	FilmID    domain.FilmID
	Role      string `json:"role"`
	Character string `json:"character"`
}
type CatalogRepo struct{ db *sql.DB }

func NewCatalogRepo(db *sql.DB) *CatalogRepo { return &CatalogRepo{db: db} }

const (
	catalogColumns = `
		p.id, btrim(concat_ws(' ', p.first_name, p.last_name)), COALESCE(p.original_name, ''),
		COALESCE(ph.external_url, '/static/' || ph.storage_key),
		COALESCE(ph.source_url, ''), COALESCE(ph.source_credit, ''),
		p.birth_date, p.death_date, COALESCE(p.birth_place, ''), COALESCE(p.biography, ''), COALESCE(p.source_url, '')`
	catalogFrom = ` FROM person AS p LEFT JOIN file AS ph ON ph.id = p.photo_file_id`

	filmPeopleQuery = `SELECT ` + catalogColumns + `, rt.code, COALESCE(fp.character_name, '')` + catalogFrom + `
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

func scanCatalogPerson(row catalogScanner) (CatalogPerson, error) {
	var p CatalogPerson
	err := row.Scan(&p.ID, &p.Name, &p.OriginalName, &p.PhotoURL, &p.PhotoSource, &p.PhotoCredit, &p.BirthDate, &p.DeathDate, &p.BirthPlace, &p.Biography, &p.SourceURL)
	return p, err
}
func (r *CatalogRepo) FilmPeople(ctx context.Context, id int64) ([]PersonCredit, error) {
	people := []PersonCredit{}
	rows, err := r.db.QueryContext(ctx, filmPeopleQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item PersonCredit
		p := &item.Person
		if err = rows.Scan(&p.ID, &p.Name, &p.OriginalName, &p.PhotoURL, &p.PhotoSource, &p.PhotoCredit, &p.BirthDate, &p.DeathDate, &p.BirthPlace, &p.Biography, &p.SourceURL, &item.Role, &item.Character); err != nil {
			return nil, err
		}
		people = append(people, item)
	}
	return people, rows.Err()
}
func (r *CatalogRepo) Get(ctx context.Context, id int64) (CatalogPerson, []FilmCredit, error) {
	person, err := scanCatalogPerson(r.db.QueryRowContext(ctx, personQuery, id))
	if errors.Is(err, sql.ErrNoRows) {
		return person, nil, domain.ErrPersonNotFound
	}
	if err != nil {
		return person, nil, err
	}
	credits := []FilmCredit{}
	rows, err := r.db.QueryContext(ctx, personFilmsQuery, id)
	if err != nil {
		return person, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var credit FilmCredit
		if err = rows.Scan(&credit.FilmID, &credit.Role, &credit.Character); err != nil {
			return person, nil, err
		}
		credits = append(credits, credit)
	}
	return person, credits, rows.Err()
}
