package entities

type CollectionID int64

type Collection struct {
	ID          CollectionID `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Slug        string       `json:"slug" db:"slug"`
	Description *string      `json:"description,omitempty" db:"description"`
	CoverURL    *string      `json:"cover_url,omitempty" db:"cover_url"`
}

type CollectionWithFilms struct {
	Collection
	Films []Film `json:"films"`
}
