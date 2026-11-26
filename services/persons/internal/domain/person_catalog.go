package domain

import (
	"context"
	"time"
)

type CatalogPerson struct {
	ID           int64
	Name         string
	OriginalName string
	PhotoURL     *string
	PhotoSource  string
	PhotoCredit  string
	BirthDate    *time.Time
	DeathDate    *time.Time
	BirthPlace   string
	Biography    string
	SourceURL    string
}

type PersonCredit struct {
	Person    CatalogPerson
	Role      string
	Character string
}

type FilmCredit struct {
	FilmID    int64
	Role      string
	Character string
}

type PersonCatalogRepository interface {
	FilmPersons(context.Context, int64) ([]PersonCredit, error)
	Get(context.Context, int64) (CatalogPerson, []FilmCredit, error)
}
