// правила работы с отзывами

package reviews

import (
	"context"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

// что прислал пользователь
type ReviewInput struct {
	Title            string
	Content          string
	ContainsSpoilers bool
}

// хранение отзывов, список методов
type ReviewRepository interface {
	Create(ctx context.Context, authorID auth.UserID, filmID entities.FilmID, input ReviewInput) (entities.Review, error)
	GetByID(ctx context.Context, id entities.ReviewID) (entities.Review, error)
	ListByFilm(ctx context.Context, filmID entities.FilmID, limit, offset int) ([]entities.Review, int, error)
	Update(ctx context.Context, id entities.ReviewID, input ReviewInput) (entities.Review, error)
	Delete(ctx context.Context, id entities.ReviewID, deletedBy auth.UserID) error
}

// проверка фильма
type FilmGetter interface {
	GetFilmByID(ctx context.Context, id entities.FilmID) (entities.Film, error)
}

type Usecase struct {
	repo  ReviewRepository
	films FilmGetter
}

// конструктор, создание нового юзкейса
func NewUsecase(repo ReviewRepository, films FilmGetter) (*Usecase, error) {
	if repo == nil {
		return nil, ErrNilRepo
	}
	if films == nil {
		return nil, ErrNilFilms
	}

	return &Usecase{repo: repo, films: films}, nil
}

// отзывы на фильм
func (uc *Usecase) ListByFilm(ctx context.Context, filmID entities.FilmID, limit, offset int) ([]entities.Review, int, error) {
	_, err := uc.films.GetFilmByID(ctx, filmID)
	if err != nil {
		return nil, 0, err
	}
	return uc.repo.ListByFilm(ctx, filmID, limit, offset)
}

// создание нового отзыва
func (uc *Usecase) Create(ctx context.Context, authorID auth.UserID, filmID entities.FilmID, input ReviewInput) (entities.Review, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)

	err := ValidateInput(input)
	if err != nil {
		return entities.Review{}, err
	}

	_, err = uc.films.GetFilmByID(ctx, filmID)
	if err != nil {
		return entities.Review{}, err
	}
	return uc.repo.Create(ctx, authorID, filmID, input)
}

// удаление отзыва
func (uc *Usecase) Delete(ctx context.Context, userID auth.UserID, id entities.ReviewID) error {
	review, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if review.AuthorID != int64(userID) {
		return ErrForbidden
	}
	return uc.repo.Delete(ctx, id, userID)
}

func (uc *Usecase) Update(ctx context.Context, userID auth.UserID, id entities.ReviewID, input ReviewInput) (entities.Review, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)

	err := ValidateInput(input)
	if err != nil {
		return entities.Review{}, err
	}

	review, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entities.Review{}, err
	}
	if review.AuthorID != int64(userID) {
		return entities.Review{}, ErrForbidden
	}
	return uc.repo.Update(ctx, id, input)
}
