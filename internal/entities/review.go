package entities

import (
	"time"
)

type ReviewID int64

type Review struct {
	ID               ReviewID  `json:"id" db:"id"`
	AuthorID         int64     `json:"author_id" db:"account_id"`
	FilmID           FilmID    `json:"film_id" db:"film_id"`
	Title            string    `json:"title" db:"title"`
	Content          string    `json:"content" db:"content"`
	ContainsSpoilers bool      `json:"contains_spoilers" db:"contains_spoilers"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}
