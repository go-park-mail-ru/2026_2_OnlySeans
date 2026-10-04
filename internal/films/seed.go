package films

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

func ptr[T any](v T) *T {
	return &v
}

var (
	genreDrama = entities.Genre{
		ID:   1,
		Name: "драма",
		Slug: "drama",
	}
	genreComedy = entities.Genre{
		ID:   2,
		Name: "комедия",
		Slug: "comedy",
	}
	genreSciFi = entities.Genre{
		ID:   3,
		Name: "фантастика",
		Slug: "sci-fi",
	}
	genreCrime = entities.Genre{
		ID:   4,
		Name: "криминал",
		Slug: "crime",
	}
	genreThriller = entities.Genre{
		ID:   5,
		Name: "триллер",
		Slug: "thriller",
	}
	genreAnimation = entities.Genre{
		ID:   6,
		Name: "мультфильм",
		Slug: "animation",
	}
	genreAdventure = entities.Genre{
		ID:   7,
		Name: "приключения",
		Slug: "adventure",
	}
	genreFamily = entities.Genre{
		ID:   8,
		Name: "семейный",
		Slug: "family",
	}
)

var seedFilms = []entities.Film{
	{
		ID:             1,
		Title:          "Побег из Шоушенка",
		OriginalTitle:  ptr("The Shawshank Redemption"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1994,
		DurationMin:    ptr[int16](142),
		AgeLimit:       16,
		Description:    ptr("Банкир Энди Дюфрейн получает пожизненный срок за убийство, которого не совершал."),
		Genres:         []entities.Genre{genreDrama},
	},
	{
		ID:             2,
		Title:          "Зелёная миля",
		OriginalTitle:  ptr("The Green Mile"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1999,
		DurationMin:    ptr[int16](189),
		AgeLimit:       16,
		Description:    ptr("Надзиратель блока смертников знакомится с заключённым, обладающим необычным даром."),
		Genres:         []entities.Genre{genreDrama, genreCrime},
	},
	{
		ID:             3,
		Title:          "Форрест Гамп",
		OriginalTitle:  ptr("Forrest Gump"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1994,
		DurationMin:    ptr[int16](142),
		AgeLimit:       12,
		Description:    ptr("История простодушного человека, который оказывается в центре главных событий эпохи."),
		Genres:         []entities.Genre{genreDrama, genreComedy},
	},
	{
		ID:             4,
		Title:          "Интерстеллар",
		OriginalTitle:  ptr("Interstellar"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 2014,
		DurationMin:    ptr[int16](169),
		AgeLimit:       16,
		Description:    ptr("Группа исследователей отправляется через червоточину в поисках нового дома для человечества."),
		Genres:         []entities.Genre{genreSciFi, genreDrama, genreAdventure},
	},
	{
		ID:             5,
		Title:          "Начало",
		OriginalTitle:  ptr("Inception"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 2010,
		DurationMin:    ptr[int16](148),
		AgeLimit:       12,
		Description:    ptr("Вор, крадущий идеи из снов, получает задание не украсть идею, а внедрить её."),
		Genres:         []entities.Genre{genreSciFi, genreThriller},
	},
	{
		ID:             6,
		Title:          "Остров проклятых",
		OriginalTitle:  ptr("Shutter Island"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 2010,
		DurationMin:    ptr[int16](138),
		AgeLimit:       18,
		Description:    ptr("Два пристава расследуют исчезновение пациентки из психиатрической клиники на острове."),
		Genres:         []entities.Genre{genreThriller, genreDrama},
	},
	{
		ID:             7,
		Title:          "Король Лев",
		OriginalTitle:  ptr("The Lion King"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1994,
		DurationMin:    ptr[int16](88),
		AgeLimit:       0,
		Description:    ptr("Львёнок Симба должен вернуть себе трон, который отнял его коварный дядя."),
		Genres:         []entities.Genre{genreAnimation, genreFamily, genreDrama},
	},
	{
		ID:             8,
		Title:          "Один дома",
		OriginalTitle:  ptr("Home Alone"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1990,
		DurationMin:    ptr[int16](103),
		AgeLimit:       0,
		Description:    ptr("Восьмилетнего Кевина случайно забывают дома, и ему приходится защищать дом от грабителей."),
		Genres:         []entities.Genre{genreComedy, genreFamily},
	},
	{
		ID:             9,
		Title:          "Иван Васильевич меняет профессию",
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 1973,
		DurationMin:    ptr[int16](88),
		AgeLimit:       6,
		Description:    ptr("Машина времени изобретателя Шурика меняет местами управдома и царя Ивана Грозного."),
		Genres:         []entities.Genre{genreComedy, genreSciFi},
	},
	{
		ID:             10,
		Title:          "Джентльмены",
		OriginalTitle:  ptr("The Gentlemen"),
		FilmType:       entities.FilmTypeMovie,
		ProductionYear: 2019,
		DurationMin:    ptr[int16](113),
		AgeLimit:       18,
		Description:    ptr("Наркобарон решает продать свой бизнес, и вокруг сделки начинается большая игра."),
		Genres:         []entities.Genre{genreCrime, genreComedy},
	},
	{
		ID:             11,
		Title:          "Шерлок",
		OriginalTitle:  ptr("Sherlock"),
		FilmType:       entities.FilmTypeSeries,
		ProductionYear: 2010,
		DurationMin:    ptr[int16](88),
		AgeLimit:       12,
		Description:    ptr("Современная версия историй о Шерлоке Холмсе и докторе Ватсоне."),
		Genres:         []entities.Genre{genreCrime, genreDrama, genreThriller},
	},
	{
		ID:             12,
		Title:          "Во все тяжкие",
		OriginalTitle:  ptr("Breaking Bad"),
		FilmType:       entities.FilmTypeSeries,
		ProductionYear: 2008,
		DurationMin:    ptr[int16](47),
		AgeLimit:       18,
		Description:    ptr("Школьный учитель химии узнаёт о смертельной болезни и начинает варить метамфетамин."),
		Genres:         []entities.Genre{genreCrime, genreDrama, genreThriller},
	},
}

type seedCollection struct {
	collection entities.Collection
	filmIDs    []entities.FilmID
}

var seedCollections = []seedCollection{
	{
		collection: entities.Collection{
			ID:          1,
			Title:       "Лучшие фильмы всех времён",
			Slug:        "best-of-all-time",
			Description: ptr("Фильмы, которые стоит посмотреть хотя бы раз."),
		},
		filmIDs: []entities.FilmID{1, 2, 3, 4, 5, 7},
	},
	{
		collection: entities.Collection{
			ID:          2,
			Title:       "Смотрим всей семьёй",
			Slug:        "family",
			Description: ptr("Добрые фильмы для вечера с близкими."),
		},
		filmIDs: []entities.FilmID{7, 8, 3, 9},
	},
	{
		collection: entities.Collection{
			ID:          3,
			Title:       "Закрученный сюжет",
			Slug:        "mind-benders",
			Description: ptr("До последней минуты не угадаешь, чем всё кончится."),
		},
		filmIDs: []entities.FilmID{5, 6, 4, 10},
	},
	{
		collection: entities.Collection{
			ID:          4,
			Title:       "Сериалы на выходные",
			Slug:        "weekend-series",
			Description: ptr("Сериалы, от которых трудно оторваться."),
		},
		filmIDs: []entities.FilmID{11, 12},
	},
}

func NewSeededService() (*InMemoryService, error) {
	service := NewInMemoryService()

	for _, film := range seedFilms {
		if err := service.AddFilm(film); err != nil {
			return nil, fmt.Errorf("seed films: %w", err)
		}
	}
	for _, seed := range seedCollections {
		if err := service.AddCollection(seed.collection, seed.filmIDs...); err != nil {
			return nil, fmt.Errorf("seed collections: %w", err)
		}
	}

	return service, nil
}
