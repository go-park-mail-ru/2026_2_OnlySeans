package persons

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type personResponseDTO struct {
	ID        domain.PersonID `json:"id"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	BirthDate *time.Time      `json:"birth_date"`
	PhotoID   *string         `json:"photo_id"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type personResponse struct {
	Person personResponseDTO `json:"person"`
}

type personsPage struct {
	Persons []personResponseDTO `json:"persons"`
	Total   int                 `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toPersonResponseDTO(person domain.Person) personResponseDTO {
	return personResponseDTO{
		ID:        person.ID,
		FirstName: person.FirstName,
		LastName:  person.LastName,
		BirthDate: person.BirthDate,
		PhotoID:   person.PhotoID,
		CreatedAt: person.CreatedAt,
		UpdatedAt: person.UpdatedAt,
	}
}

func toPersonResponseDTOs(persons []domain.Person) []personResponseDTO {
	result := make([]personResponseDTO, 0, len(persons))

	for _, person := range persons {
		result = append(result, toPersonResponseDTO(person))
	}

	return result
}
