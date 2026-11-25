package domain

type CollectionID int64

type Collection struct {
	ID          CollectionID
	Title       string
	Slug        string
	Description *string
	CoverURL    *string
}
