package auth

import (
	"time"
)

type User struct {
	ID           int64      `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`
	Gender       *string    `json:"gender,omitempty" db:"gender"`
	BirthDate    *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Bio          *string    `json:"bio,omitempty" db:"bio"`
	Role         string     `json:"role" db:"role"`
	IsPrivate    bool       `json:"is_private" db:"is_private"`
	AvatarURL    *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type Film struct {
	ID            int64     `json:"id" db:"id"`
	Title         string    `json:"title" db:"title"`
	OriginalTitle *string   `json:"original_title,omitempty" db:"original_title"`
	FilmType      string    `json:"film_type" db:"film_type"`
	ReleaseYear   int16     `json:"release_year" db:"release_year"`
	DurationMin   *int16    `json:"duration_min,omitempty" db:"duration_min"`
	AgeLimit      int16     `json:"age_limit" db:"age_limit"`
	Description   *string   `json:"description,omitempty" db:"description"`
	PosterURL     *string   `json:"poster_url,omitempty" db:"poster_url"`
	TrailerURL    *string   `json:"trailer_url,omitempty" db:"trailer_url"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Genre struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmGenre struct {
	FilmID    int64     `json:"film_id" db:"film_id"`
	GenreID   int64     `json:"genre_id" db:"genre_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Country struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ISOCode   string    `json:"iso_code" db:"iso_code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmCountry struct {
	FilmID    int64     `json:"film_id" db:"film_id"`
	CountryID int64     `json:"country_id" db:"country_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type SimilarFilm struct {
	FilmID        int64     `json:"film_id" db:"film_id"`
	SimilarFilmID int64     `json:"similar_film_id" db:"similar_film_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type FilmRelease struct {
	FilmID      int64     `json:"film_id" db:"film_id"`
	CountryID   int64     `json:"country_id" db:"country_id"`
	ReleaseType string    `json:"release_type" db:"release_type"`
	ReleaseDate time.Time `json:"release_date" db:"release_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Season struct {
	FilmID       int64     `json:"film_id" db:"film_id"`
	SeasonNumber int16     `json:"season_number" db:"season_number"`
	Title        *string   `json:"title,omitempty" db:"title"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Episode struct {
	FilmID        int64      `json:"film_id" db:"film_id"`
	SeasonNumber  int16      `json:"season_number" db:"season_number"`
	EpisodeNumber int16      `json:"episode_number" db:"episode_number"`
	Title         string     `json:"title" db:"title"`
	DurationMin   *int16     `json:"duration_min,omitempty" db:"duration_min"`
	ReleaseDate   *time.Time `json:"release_date,omitempty" db:"release_date"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type Person struct {
	ID        int64      `json:"id" db:"id"`
	FirstName string     `json:"first_name" db:"first_name"`
	LastName  string     `json:"last_name" db:"last_name"`
	BirthDate *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	PhotoURL  *string    `json:"photo_url,omitempty" db:"photo_url"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

type RoleType struct {
	ID        int64     `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmPerson struct {
	ID            int64     `json:"id" db:"id"`
	FilmID        int64     `json:"film_id" db:"film_id"`
	PersonID      int64     `json:"person_id" db:"person_id"`
	RoleTypeID    int64     `json:"role_type_id" db:"role_type_id"`
	CharacterName *string   `json:"character_name,omitempty" db:"character_name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type FilmRating struct {
	AccountID int64     `json:"account_id" db:"account_id"`
	FilmID    int64     `json:"film_id" db:"film_id"`
	Score     int16     `json:"score" db:"score"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmRatingHistory struct {
	ID        int64     `json:"id" db:"id"`
	AccountID int64     `json:"account_id" db:"account_id"`
	FilmID    int64     `json:"film_id" db:"film_id"`
	OldScore  *int16    `json:"old_score,omitempty" db:"old_score"`
	NewScore  int16     `json:"new_score" db:"new_score"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Review struct {
	ID               int64     `json:"id" db:"id"`
	AccountID        int64     `json:"account_id" db:"account_id"`
	FilmID           int64     `json:"film_id" db:"film_id"`
	Title            string    `json:"title" db:"title"`
	Content          string    `json:"content" db:"content"`
	ContainsSpoilers bool      `json:"contains_spoilers" db:"contains_spoilers"`
	IsApproved       bool      `json:"is_approved" db:"is_approved"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type Folder struct {
	ID         int64     `json:"id" db:"id"`
	AccountID  int64     `json:"account_id" db:"account_id"`
	Title      string    `json:"title" db:"title"`
	FolderType string    `json:"folder_type" db:"folder_type"`
	IsPrivate  bool      `json:"is_private" db:"is_private"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type FolderItem struct {
	ID        int64     `json:"id" db:"id"`
	FolderID  int64     `json:"folder_id" db:"folder_id"`
	FilmID    *int64    `json:"film_id,omitempty" db:"film_id"`
	PersonID  *int64    `json:"person_id,omitempty" db:"person_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ReleaseSubscription struct {
	AccountID int64     `json:"account_id" db:"account_id"`
	FilmID    int64     `json:"film_id" db:"film_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Collection struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Slug        string    `json:"slug" db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	CoverURL    *string   `json:"cover_url,omitempty" db:"cover_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CollectionFilm struct {
	CollectionID int64     `json:"collection_id" db:"collection_id"`
	FilmID       int64     `json:"film_id" db:"film_id"`
	Position     int16     `json:"position" db:"position"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Notification struct {
	ID        int64     `json:"id" db:"id"`
	AccountID int64     `json:"account_id" db:"account_id"`
	Message   string    `json:"message" db:"message"`
	IsRead    bool      `json:"is_read" db:"is_read"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmDiscussion struct {
	ID        int64     `json:"id" db:"id"`
	FilmID    int64     `json:"film_id" db:"film_id"`
	IsClosed  bool      `json:"is_closed" db:"is_closed"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type DiscussionMessage struct {
	ID           int64     `json:"id" db:"id"`
	DiscussionID int64     `json:"discussion_id" db:"discussion_id"`
	AccountID    int64     `json:"account_id" db:"account_id"`
	MessageText  string    `json:"message_text" db:"message_text"`
	IsDeleted    bool      `json:"is_deleted" db:"is_deleted"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
