package films

type FilmType string

const (
	FilmTypeMovie  FilmType = "movie"
	FilmTypeSeries FilmType = "series"
)

type Genre struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
	Slug string `json:"slug" db:"slug"`
}

type Film struct {
	ID            int64    `json:"id" db:"id"`
	Title         string   `json:"title" db:"title"`
	OriginalTitle *string  `json:"original_title,omitempty" db:"original_title"`
	FilmType      FilmType `json:"film_type" db:"film_type"`
	ReleaseYear   int16    `json:"release_year" db:"release_year"`
	DurationMin   *int16   `json:"duration_min,omitempty" db:"duration_min"`
	AgeLimit      int16    `json:"age_limit" db:"age_limit"`
	Description   *string  `json:"description,omitempty" db:"description"`
	PosterURL     *string  `json:"poster_url,omitempty" db:"poster_url"`
	TrailerURL    *string  `json:"trailer_url,omitempty" db:"trailer_url"`
	Genres        []Genre  `json:"genres"`
}

type Collection struct {
	ID          int64   `json:"id" db:"id"`
	Title       string  `json:"title" db:"title"`
	Slug        string  `json:"slug" db:"slug"`
	Description *string `json:"description,omitempty" db:"description"`
	CoverURL    *string `json:"cover_url,omitempty" db:"cover_url"`
}

type CollectionWithFilms struct {
	Collection
	Films []Film `json:"films"`
}

type FilmsPage struct {
	Films  []Film `json:"films"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
