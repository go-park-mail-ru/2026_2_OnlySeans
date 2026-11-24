package persons

import (
	"encoding/json"
	"time"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

//easyjson:json
type catalogPersonDTO struct {
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

//easyjson:json
type personCreditDTO struct {
	Person    catalogPersonDTO `json:"person"`
	Role      string           `json:"role"`
	Character string           `json:"character"`
}

//easyjson:json
type filmPersonsResponse struct {
	People []personCreditDTO `json:"people"`
}

//easyjson:json
type catalogFilmDTO struct {
	Film      json.RawMessage `json:"film"`
	Role      string          `json:"role"`
	Character string          `json:"character"`
}

//easyjson:json
type personCatalogResponse struct {
	Person catalogPersonDTO `json:"person"`
	Films  []catalogFilmDTO `json:"films"`
}

func toCatalogPersonDTO(person *pb.CatalogPerson) catalogPersonDTO {
	return catalogPersonDTO{
		ID:           person.GetId(),
		Name:         person.GetName(),
		OriginalName: person.GetOriginalName(),
		PhotoURL:     person.PhotoUrl,
		PhotoSource:  person.GetPhotoSource(),
		PhotoCredit:  person.GetPhotoCredit(),
		BirthDate:    toOptionalTime(person.GetBirthDate()),
		DeathDate:    toOptionalTime(person.GetDeathDate()),
		BirthPlace:   person.GetBirthPlace(),
		Biography:    person.GetBiography(),
		SourceURL:    person.GetSourceUrl(),
	}
}
