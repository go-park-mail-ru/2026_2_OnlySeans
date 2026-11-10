package domain

import (
	"context"
	"time"
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
	FilmID    FilmID
	Role      string `json:"role"`
	Character string `json:"character"`
}
type PersonCatalogRepository interface {
	FilmPersons(context.Context, int64) ([]PersonCredit, error)
	Get(context.Context, int64) (CatalogPerson, []FilmCredit, error)
}
