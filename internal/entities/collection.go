package entities

type CollectionID int64

type Collection struct {
	ID          CollectionID `db:"id"`
	Title       string       `db:"title"`
	Slug        string       `db:"slug"`
	Description *string      `db:"description"`
	CoverURL    *string      `db:"cover_url"`
}
