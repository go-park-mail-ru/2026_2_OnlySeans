package persons

import (
	"time"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//easyjson:json
type personResponseDTO struct {
	ID        int64      `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	BirthDate *time.Time `json:"birth_date"`
	PhotoID   *string    `json:"photo_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

//easyjson:json
type personResponse struct {
	Person personResponseDTO `json:"person"`
}

//easyjson:json
type personsPage struct {
	Persons []personResponseDTO `json:"persons"`
	Total   int64               `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}

//easyjson:json
type createPersonResponse struct {
	ID int64 `json:"id"`
}

func toPersonResponseDTO(person *pb.Person) personResponseDTO {
	return personResponseDTO{
		ID:        person.GetId(),
		FirstName: person.GetFirstName(),
		LastName:  person.GetLastName(),
		BirthDate: toOptionalTime(person.GetBirthDate()),
		PhotoID:   person.PhotoId,
		CreatedAt: person.GetCreatedAt().AsTime(),
		UpdatedAt: person.GetUpdatedAt().AsTime(),
	}
}

func toPersonResponseDTOs(persons []*pb.Person) []personResponseDTO {
	result := make([]personResponseDTO, len(persons))
	for i, person := range persons {
		result[i] = toPersonResponseDTO(person)
	}
	return result
}

func toOptionalTime(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}

	date := value.AsTime()
	return &date
}
