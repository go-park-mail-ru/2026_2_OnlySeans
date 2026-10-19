// храниище, держит отзывы в оп (временно)

package reviews

import (
	"context"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

// ключ для проверки дубликатов
type reviewKey struct {
	authorID int64
	filmID   entities.FilmID
}

// структура хранилища
type InMemoryReviewRepo struct {
	mu           sync.Mutex                            //замок
	byID         map[entities.ReviewID]entities.Review //хранилище
	byAuthorFilm map[reviewKey]entities.ReviewID       //кто на какой фильм писал
	nextID       entities.ReviewID                     //следующий id
}

// конструктор, при старте создаем хранилище
func NewInMemoryReviewRepo() *InMemoryReviewRepo {
	return &InMemoryReviewRepo{
		byID:         make(map[entities.ReviewID]entities.Review),
		byAuthorFilm: make(map[reviewKey]entities.ReviewID),
		nextID:       1, //иначе 0
	}
}

var _ ReviewRepository = (*InMemoryReviewRepo)(nil) //для проверки при сборкеб тк иначем можем принять неверные данные

func (r *InMemoryReviewRepo) Create(ctx context.Context, authorID auth.UserID, filmID entities.FilmID, input ReviewInput) (entities.Review, error) {
	r.mu.Lock()         //закрыть если два одновременно
	defer r.mu.Unlock() //открыть при выходе из метода

	key := reviewKey{authorID: int64(authorID), filmID: filmID} //ключ для проверки дубликата
	if _, exists := r.byAuthorFilm[key]; exists {               //если уже есть отзыв от этого автора на этот фильм
		return entities.Review{}, ErrReviewExists
	}

	//сборка отзыва
	now := time.Now()
	review := entities.Review{
		ID:               r.nextID,
		AuthorID:         int64(authorID),
		FilmID:           filmID,
		Title:            input.Title,
		Content:          input.Content,
		ContainsSpoilers: input.ContainsSpoilers,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	r.byID[review.ID] = review //помещаем в хранилище
	r.nextID++
	r.byAuthorFilm[key] = review.ID //для проверки дубликата, автор уже писал на этот фильм
	return review, nil
}

func (r *InMemoryReviewRepo) GetByID(ctx context.Context, id entities.ReviewID) (entities.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id] //достаем отзыв
	if !exists {
		return entities.Review{}, ErrReviewNotFound
	}

	return review, nil
}

func (r *InMemoryReviewRepo) Update(ctx context.Context, id entities.ReviewID, input ReviewInput) (entities.Review, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id] //достаем отзыв
	if !exists {
		return entities.Review{}, ErrReviewNotFound
	}

	//меняем
	review.Title = input.Title
	review.Content = input.Content
	review.ContainsSpoilers = input.ContainsSpoilers
	review.UpdatedAt = time.Now()

	//возвращаем
	r.byID[id] = review //до этого копия
	return review, nil
}

// все отзывыб но отдает ток запрошенную страницу
func (r *InMemoryReviewRepo) ListByFilm(ctx context.Context, filmID entities.FilmID, limit, offset int) ([]entities.Review, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	matched := make([]entities.Review, 0)

	//тк словарь перебирается в случайном порядке, то по номерам
	for id := r.nextID - 1; id >= 1; id-- { //от последнего к первому
		review, exists := r.byID[id]
		if exists && review.FilmID == filmID { //exists тк номера удаленных отзывов остаются
			matched = append(matched, review)
		}
	}

	total := len(matched)
	start := min(offset, total) //с какокго начинать(не дальше конца)
	end := min(start+limit, total)

	return matched[start:end], total, nil

}

func (r *InMemoryReviewRepo) Delete(ctx context.Context, id entities.ReviewID, deletedBy auth.UserID) error {
	//uscase проверяет мой ли отзыв

	r.mu.Lock()
	defer r.mu.Unlock()

	review, exists := r.byID[id]
	if !exists {
		return ErrReviewNotFound
	}

	delete(r.byID, id) //основной словарь

	//тк create проверяет по byAuthorFilm то хранилище считает что он есть
	key := reviewKey{authorID: review.AuthorID, filmID: review.FilmID}
	delete(r.byAuthorFilm, key)

	return nil
}
