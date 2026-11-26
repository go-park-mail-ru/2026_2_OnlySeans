package domain

import (
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	MinFirstNameLen = 1
	MaxFirstNameLen = 128
	MinLastNameLen  = 1
	MaxLastNameLen  = 128
)

var LowerLimitBirthDate = time.Date(1800, time.January, 1, 0, 0, 0, 0, time.UTC)

type PersonID int64

type Person struct {
	ID        PersonID   `db:"id"`
	FirstName string     `db:"first_name"`
	LastName  string     `db:"last_name"`
	BirthDate *time.Time `db:"birth_date"`
	PhotoID   *string    `db:"photo_file_id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type PersonInput struct {
	FirstName string
	LastName  string
	BirthDate *time.Time
}

func (in *PersonInput) Validate() error {
	firstNameLen := utf8.RuneCountInString(in.FirstName)
	if firstNameLen < MinFirstNameLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidFirstName, MinFirstNameLen)
	}
	if firstNameLen > MaxFirstNameLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidFirstName, MaxFirstNameLen)
	}

	lastNameLen := utf8.RuneCountInString(in.LastName)
	if lastNameLen < MinLastNameLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidLastName, MinLastNameLen)
	}
	if lastNameLen > MaxLastNameLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidLastName, MaxLastNameLen)
	}

	birthDate := in.BirthDate
	if birthDate != nil {
		switch {
		case birthDate.After(time.Now().UTC()):
			return fmt.Errorf("%w: must not be in the future", ErrInvalidBirthDate)
		case !birthDate.After(LowerLimitBirthDate):
			return fmt.Errorf("%w: must be after %s", ErrInvalidBirthDate, LowerLimitBirthDate.Format("2006-01-02"))
		}
	}

	return nil
}
