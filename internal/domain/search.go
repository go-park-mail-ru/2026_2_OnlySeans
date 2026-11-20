package domain

type SearchFilm struct {
	ID            FilmID
	Title         string
	OriginalTitle *string
	ReleaseYear   int16
	FilmType      FilmType
	PosterURL     *string

	PosterStorageKey *string
}

type SearchPerson struct {
	ID       PersonID
	Name     string
	PhotoURL *string
}
