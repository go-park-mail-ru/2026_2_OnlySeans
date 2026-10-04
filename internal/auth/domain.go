package auth

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type UserID int64
type AccountID int64
type RoleID int64
type PermissionID int64
type FileID int64
type FilmID int64
type GenreID int64
type CountryID int64
type PersonID int64
type RoleTypeID int64
type FilmPersonID int64
type FilmRatingHistoryID int64
type ReviewID int64
type FolderID int64
type FolderItemID int64
type CollectionID int64
type NotificationID int64
type FilmDiscussionID int64
type DiscussionMessageID int64
type ModerationTaskID int64
type AccountBanID int64

type TSTZRange struct {
	Lower          *time.Time `json:"lower,omitempty"`
	Upper          *time.Time `json:"upper,omitempty"`
	LowerInclusive bool       `json:"lower_inclusive"`
	UpperInclusive bool       `json:"upper_inclusive"`
}

func (r TSTZRange) Value() (driver.Value, error) {
	lower, err := formatRangeBound(r.Lower)
	if err != nil {
		return nil, fmt.Errorf("format lower bound: %w", err)
	}
	upper, err := formatRangeBound(r.Upper)
	if err != nil {
		return nil, fmt.Errorf("format upper bound: %w", err)
	}

	open := '('
	if r.Lower != nil && r.LowerInclusive {
		open = '['
	}
	close := ')'
	if r.Upper != nil && r.UpperInclusive {
		close = ']'
	}

	return string(open) + lower + "," + upper + string(close), nil
}

func (r *TSTZRange) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("cannot scan NULL into TSTZRange")
	}

	var raw string
	switch value := value.(type) {
	case string:
		raw = value
	case []byte:
		raw = string(value)
	default:
		return fmt.Errorf("cannot scan %T into TSTZRange", value)
	}

	if raw == "empty" {
		return fmt.Errorf("empty tstzrange is not supported")
	}
	if len(raw) < 3 || (raw[0] != '[' && raw[0] != '(') ||
		(raw[len(raw)-1] != ']' && raw[len(raw)-1] != ')') {
		return fmt.Errorf("invalid tstzrange value %q", raw)
	}

	lowerText, upperText, err := splitRangeBounds(raw[1 : len(raw)-1])
	if err != nil {
		return err
	}

	lower, err := parseRangeBound(lowerText)
	if err != nil {
		return fmt.Errorf("parse lower bound: %w", err)
	}
	upper, err := parseRangeBound(upperText)
	if err != nil {
		return fmt.Errorf("parse upper bound: %w", err)
	}

	r.Lower = lower
	r.Upper = upper
	r.LowerInclusive = raw[0] == '[' && lower != nil
	r.UpperInclusive = raw[len(raw)-1] == ']' && upper != nil
	return nil
}

func formatRangeBound(bound *time.Time) (string, error) {
	if bound == nil {
		return "", nil
	}
	return strconv.Quote(bound.Format(time.RFC3339Nano)), nil
}

func splitRangeBounds(raw string) (string, string, error) {
	quoted := false
	escaped := false
	for i, char := range raw {
		if escaped {
			escaped = false
			continue
		}
		switch {
		case char == '\\' && quoted:
			escaped = true
		case char == '"':
			quoted = !quoted
		case char == ',' && !quoted:
			return raw[:i], raw[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid tstzrange bounds %q", raw)
}

func parseRangeBound(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-infinity" || raw == "infinity" {
		return nil, nil
	}
	if strings.HasPrefix(raw, `"`) {
		unquoted, err := strconv.Unquote(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid quoted bound %q: %w", raw, err)
		}
		raw = unquoted
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999-07",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05-07",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("unsupported timestamp %q", raw)
}

type Role struct {
	ID        RoleID    `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	ID        PermissionID `json:"id" db:"id"`
	Code      string       `json:"code" db:"code"`
	Title     string       `json:"title" db:"title"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
}

type RolePermission struct {
	RoleID       RoleID       `json:"role_id" db:"role_id"`
	PermissionID PermissionID `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

type Account struct {
	ID           AccountID  `json:"id" db:"id"`
	RoleID       RoleID     `json:"role_id" db:"role_id"`
	Email        string     `json:"email" db:"email"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`
	Gender       *string    `json:"gender,omitempty" db:"gender"`
	BirthDate    *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	Bio          *string    `json:"bio,omitempty" db:"bio"`
	AvatarFileID *FileID    `json:"avatar_file_id,omitempty" db:"avatar_file_id"`
	IsPrivate    bool       `json:"is_private" db:"is_private"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type User struct {
	ID           UserID `json:"id" db:"id"`
	Email        string `json:"email" db:"email"`
	Username     string `json:"username" db:"username"`
	PasswordHash string `json:"-" db:"password_hash"`
}

type File struct {
	ID          FileID    `json:"id" db:"id"`
	StorageKey  *string   `json:"storage_key,omitempty" db:"storage_key"`
	ExternalURL *string   `json:"external_url,omitempty" db:"external_url"`
	MimeType    string    `json:"mime_type" db:"mime_type"`
	SizeBytes   *int64    `json:"size_bytes,omitempty" db:"size_bytes"`
	CreatedBy   *UserID   `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Film struct {
	ID             FilmID    `json:"id" db:"id"`
	Title          string    `json:"title" db:"title"`
	OriginalTitle  *string   `json:"original_title,omitempty" db:"original_title"`
	FilmType       string    `json:"film_type" db:"film_type"`
	ProductionYear int16     `json:"production_year" db:"production_year"`
	DurationMin    *int16    `json:"duration_min,omitempty" db:"duration_min"`
	AgeLimit       int16     `json:"age_limit" db:"age_limit"`
	Description    *string   `json:"description,omitempty" db:"description"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type FilmFile struct {
	FilmID    FilmID    `json:"film_id" db:"film_id"`
	FileID    FileID    `json:"file_id" db:"file_id"`
	FileRole  string    `json:"file_role" db:"file_role"`
	SortOrder int16     `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func NewFilmFile(filmID FilmID, fileID FileID, role string) FilmFile {
	return FilmFile{FilmID: filmID, FileID: fileID, FileRole: role, SortOrder: 1}
}

type Genre struct {
	ID        GenreID   `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmGenre struct {
	FilmID    FilmID    `json:"film_id" db:"film_id"`
	GenreID   GenreID   `json:"genre_id" db:"genre_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Country struct {
	ID        CountryID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ISOCode   string    `json:"iso_code" db:"iso_code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmCountry struct {
	FilmID    FilmID    `json:"film_id" db:"film_id"`
	CountryID CountryID `json:"country_id" db:"country_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type SimilarFilm struct {
	FilmID        FilmID    `json:"film_id" db:"film_id"`
	SimilarFilmID FilmID    `json:"similar_film_id" db:"similar_film_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type FilmRelease struct {
	FilmID      FilmID    `json:"film_id" db:"film_id"`
	CountryID   CountryID `json:"country_id" db:"country_id"`
	ReleaseType string    `json:"release_type" db:"release_type"`
	ReleaseDate time.Time `json:"release_date" db:"release_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Season struct {
	FilmID       FilmID    `json:"film_id" db:"film_id"`
	FilmType     string    `json:"film_type" db:"film_type"`
	SeasonNumber int16     `json:"season_number" db:"season_number"`
	Title        *string   `json:"title,omitempty" db:"title"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func NewSeason(filmID FilmID, seasonNumber int16) Season {
	return Season{FilmID: filmID, FilmType: "series", SeasonNumber: seasonNumber}
}

type Episode struct {
	FilmID        FilmID     `json:"film_id" db:"film_id"`
	SeasonNumber  int16      `json:"season_number" db:"season_number"`
	EpisodeNumber int16      `json:"episode_number" db:"episode_number"`
	Title         string     `json:"title" db:"title"`
	DurationMin   *int16     `json:"duration_min,omitempty" db:"duration_min"`
	ReleaseDate   *time.Time `json:"release_date,omitempty" db:"release_date"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type Person struct {
	ID          PersonID   `json:"id" db:"id"`
	FirstName   string     `json:"first_name" db:"first_name"`
	LastName    string     `json:"last_name" db:"last_name"`
	BirthDate   *time.Time `json:"birth_date,omitempty" db:"birth_date"`
	PhotoFileID *FileID    `json:"photo_file_id,omitempty" db:"photo_file_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type RoleType struct {
	ID        RoleTypeID `json:"id" db:"id"`
	Code      string     `json:"code" db:"code"`
	Title     string     `json:"title" db:"title"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

type FilmPerson struct {
	ID            FilmPersonID `json:"id" db:"id"`
	FilmID        FilmID       `json:"film_id" db:"film_id"`
	PersonID      PersonID     `json:"person_id" db:"person_id"`
	RoleTypeID    RoleTypeID   `json:"role_type_id" db:"role_type_id"`
	CharacterName *string      `json:"character_name,omitempty" db:"character_name"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

type FilmRating struct {
	AccountID UserID    `json:"account_id" db:"account_id"`
	FilmID    FilmID    `json:"film_id" db:"film_id"`
	Score     int16     `json:"score" db:"score"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FilmRatingHistory struct {
	ID        FilmRatingHistoryID `json:"id" db:"id"`
	AccountID UserID              `json:"account_id" db:"account_id"`
	FilmID    FilmID              `json:"film_id" db:"film_id"`
	OldScore  *int16              `json:"old_score,omitempty" db:"old_score"`
	NewScore  int16               `json:"new_score" db:"new_score"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
}

type Review struct {
	ID               ReviewID   `json:"id" db:"id"`
	AccountID        UserID     `json:"account_id" db:"account_id"`
	FilmID           FilmID     `json:"film_id" db:"film_id"`
	Title            string     `json:"title" db:"title"`
	Content          string     `json:"content" db:"content"`
	ContainsSpoilers bool       `json:"contains_spoilers" db:"contains_spoilers"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedBy        *UserID    `json:"deleted_by,omitempty" db:"deleted_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type Folder struct {
	ID         FolderID  `json:"id" db:"id"`
	AccountID  UserID    `json:"account_id" db:"account_id"`
	Title      string    `json:"title" db:"title"`
	FolderType string    `json:"folder_type" db:"folder_type"`
	IsPrivate  bool      `json:"is_private" db:"is_private"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

func NewFolder(accountID UserID, title, folderType string) Folder {
	return Folder{
		AccountID:  accountID,
		Title:      title,
		FolderType: folderType,
		IsPrivate:  true,
	}
}

type FolderItem struct {
	ID        FolderItemID `json:"id" db:"id"`
	FolderID  FolderID     `json:"folder_id" db:"folder_id"`
	FilmID    *FilmID      `json:"film_id,omitempty" db:"film_id"`
	PersonID  *PersonID    `json:"person_id,omitempty" db:"person_id"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
}

type ReleaseSubscription struct {
	AccountID UserID    `json:"account_id" db:"account_id"`
	FilmID    FilmID    `json:"film_id" db:"film_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Collection struct {
	ID          CollectionID `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Slug        string       `json:"slug" db:"slug"`
	Description *string      `json:"description,omitempty" db:"description"`
	CoverFileID *FileID      `json:"cover_file_id,omitempty" db:"cover_file_id"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type CollectionFilm struct {
	CollectionID CollectionID `json:"collection_id" db:"collection_id"`
	FilmID       FilmID       `json:"film_id" db:"film_id"`
	Position     int16        `json:"position" db:"position"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

type Notification struct {
	ID        NotificationID `json:"id" db:"id"`
	AccountID UserID         `json:"account_id" db:"account_id"`
	Message   string         `json:"message" db:"message"`
	IsRead    bool           `json:"is_read" db:"is_read"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

type FilmDiscussion struct {
	ID        FilmDiscussionID `json:"id" db:"id"`
	FilmID    FilmID           `json:"film_id" db:"film_id"`
	IsClosed  bool             `json:"is_closed" db:"is_closed"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
}

type DiscussionMessage struct {
	ID           DiscussionMessageID `json:"id" db:"id"`
	DiscussionID FilmDiscussionID    `json:"discussion_id" db:"discussion_id"`
	AccountID    UserID              `json:"account_id" db:"account_id"`
	MessageText  string              `json:"message_text" db:"message_text"`
	DeletedAt    *time.Time          `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedBy    *UserID             `json:"deleted_by,omitempty" db:"deleted_by"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" db:"updated_at"`
}

type ModerationTask struct {
	ID         ModerationTaskID     `json:"id" db:"id"`
	ReviewID   *ReviewID            `json:"review_id,omitempty" db:"review_id"`
	MessageID  *DiscussionMessageID `json:"message_id,omitempty" db:"message_id"`
	Status     string               `json:"status" db:"status"`
	Source     string               `json:"source" db:"source"`
	ModelName  *string              `json:"model_name,omitempty" db:"model_name"`
	ModelScore *float64             `json:"model_score,omitempty" db:"model_score"`
	DecidedBy  *UserID              `json:"decided_by,omitempty" db:"decided_by"`
	Reason     *string              `json:"reason,omitempty" db:"reason"`
	DecidedAt  *time.Time           `json:"decided_at,omitempty" db:"decided_at"`
	CreatedAt  time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at" db:"updated_at"`
}

type AccountBan struct {
	ID           AccountBanID `json:"id" db:"id"`
	AccountID    UserID       `json:"account_id" db:"account_id"`
	ModeratorID  UserID       `json:"moderator_id" db:"moderator_id"`
	Reason       string       `json:"reason" db:"reason"`
	ActiveDuring TSTZRange    `json:"active_during" db:"active_during"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}
