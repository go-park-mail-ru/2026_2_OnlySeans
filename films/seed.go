package films

import "fmt"

func ptr[T any](v T) *T {
	return &v
}

var (
	genreDrama     = Genre{ID: 1, Name: "драма", Slug: "drama"}
	genreComedy    = Genre{ID: 2, Name: "комедия", Slug: "comedy"}
	genreSciFi     = Genre{ID: 3, Name: "фантастика", Slug: "sci-fi"}
	genreCrime     = Genre{ID: 4, Name: "криминал", Slug: "crime"}
	genreThriller  = Genre{ID: 5, Name: "триллер", Slug: "thriller"}
	genreAnimation = Genre{ID: 6, Name: "мультфильм", Slug: "animation"}
	genreAdventure = Genre{ID: 7, Name: "приключения", Slug: "adventure"}
	genreFamily    = Genre{ID: 8, Name: "семейный", Slug: "family"}
)

var seedFilms = []Film{
	{
		ID: 1, Title: "Побег из Шоушенка", OriginalTitle: ptr("The Shawshank Redemption"),
		FilmType: FilmTypeMovie, ReleaseYear: 1994, DurationMin: ptr[int16](142), AgeLimit: 16,
		Description: ptr("Банкир Энди Дюфрейн получает пожизненный срок за убийство, которого не совершал."),
		Genres:      []Genre{genreDrama},
	},
	{
		ID: 2, Title: "Зелёная миля", OriginalTitle: ptr("The Green Mile"),
		FilmType: FilmTypeMovie, ReleaseYear: 1999, DurationMin: ptr[int16](189), AgeLimit: 16,
		Description: ptr("Надзиратель блока смертников знакомится с заключённым, обладающим необычным даром."),
		Genres:      []Genre{genreDrama, genreCrime},
	},
	{
		ID: 3, Title: "Форрест Гамп", OriginalTitle: ptr("Forrest Gump"),
		FilmType: FilmTypeMovie, ReleaseYear: 1994, DurationMin: ptr[int16](142), AgeLimit: 12,
		Description: ptr("История простодушного человека, который оказывается в центре главных событий эпохи."),
		Genres:      []Genre{genreDrama, genreComedy},
	},
	{
		ID: 4, Title: "Интерстеллар", OriginalTitle: ptr("Interstellar"),
		FilmType: FilmTypeMovie, ReleaseYear: 2014, DurationMin: ptr[int16](169), AgeLimit: 16,
		Description: ptr("Группа исследователей отправляется через червоточину в поисках нового дома для человечества."),
		Genres:      []Genre{genreSciFi, genreDrama, genreAdventure},
	},
	{
		ID: 5, Title: "Начало", OriginalTitle: ptr("Inception"),
		FilmType: FilmTypeMovie, ReleaseYear: 2010, DurationMin: ptr[int16](148), AgeLimit: 12,
		Description: ptr("Вор, крадущий идеи из снов, получает задание не украсть идею, а внедрить её."),
		Genres:      []Genre{genreSciFi, genreThriller},
	},
	{
		ID: 6, Title: "Остров проклятых", OriginalTitle: ptr("Shutter Island"),
		FilmType: FilmTypeMovie, ReleaseYear: 2010, DurationMin: ptr[int16](138), AgeLimit: 18,
		Description: ptr("Два пристава расследуют исчезновение пациентки из психиатрической клиники на острове."),
		Genres:      []Genre{genreThriller, genreDrama},
	},
	{
		ID: 7, Title: "Король Лев", OriginalTitle: ptr("The Lion King"),
		FilmType: FilmTypeMovie, ReleaseYear: 1994, DurationMin: ptr[int16](88), AgeLimit: 0,
		Description: ptr("Львёнок Симба должен вернуть себе трон, который отнял его коварный дядя."),
		Genres:      []Genre{genreAnimation, genreFamily, genreDrama},
	},
	{
		ID: 8, Title: "Один дома", OriginalTitle: ptr("Home Alone"),
		FilmType: FilmTypeMovie, ReleaseYear: 1990, DurationMin: ptr[int16](103), AgeLimit: 0,
		Description: ptr("Восьмилетнего Кевина случайно забывают дома, и ему приходится защищать дом от грабителей."),
		Genres:      []Genre{genreComedy, genreFamily},
	},
	{
		ID: 9, Title: "Иван Васильевич меняет профессию",
		FilmType: FilmTypeMovie, ReleaseYear: 1973, DurationMin: ptr[int16](88), AgeLimit: 6,
		Description: ptr("Машина времени изобретателя Шурика меняет местами управдома и царя Ивана Грозного."),
		Genres:      []Genre{genreComedy, genreSciFi},
	},
	{
		ID: 10, Title: "Джентльмены", OriginalTitle: ptr("The Gentlemen"),
		FilmType: FilmTypeMovie, ReleaseYear: 2019, DurationMin: ptr[int16](113), AgeLimit: 18,
		Description: ptr("Наркобарон решает продать свой бизнес, и вокруг сделки начинается большая игра."),
		Genres:      []Genre{genreCrime, genreComedy},
	},
	{
		ID: 11, Title: "Шерлок", OriginalTitle: ptr("Sherlock"),
		FilmType: FilmTypeSeries, ReleaseYear: 2010, DurationMin: ptr[int16](88), AgeLimit: 12,
		Description: ptr("Современная версия историй о Шерлоке Холмсе и докторе Ватсоне."),
		Genres:      []Genre{genreCrime, genreDrama, genreThriller},
	},
	{
		ID: 12, Title: "Во все тяжкие", OriginalTitle: ptr("Breaking Bad"),
		FilmType: FilmTypeSeries, ReleaseYear: 2008, DurationMin: ptr[int16](47), AgeLimit: 18,
		Description: ptr("Школьный учитель химии узнаёт о смертельной болезни и начинает варить метамфетамин."),
		Genres:      []Genre{genreCrime, genreDrama, genreThriller},
	},
}

var seedCollections = []struct {
	collection Collection
	filmIDs    []int64
}{
	{
		collection: Collection{
			ID: 1, Title: "Лучшие фильмы всех времён", Slug: "best-of-all-time",
			Description: ptr("Фильмы, которые стоит посмотреть хотя бы раз."),
		},
		filmIDs: []int64{1, 2, 3, 4, 5, 7},
	},
	{
		collection: Collection{
			ID: 2, Title: "Смотрим всей семьёй", Slug: "family",
			Description: ptr("Добрые фильмы для вечера с близкими."),
		},
		filmIDs: []int64{7, 8, 3, 9},
	},
	{
		collection: Collection{
			ID: 3, Title: "Закрученный сюжет", Slug: "mind-benders",
			Description: ptr("До последней минуты не угадаешь, чем всё кончится."),
		},
		filmIDs: []int64{5, 6, 4, 10},
	},
	{
		collection: Collection{
			ID: 4, Title: "Сериалы на выходные", Slug: "weekend-series",
			Description: ptr("Сериалы, от которых трудно оторваться."),
		},
		filmIDs: []int64{11, 12},
	},
}

func NewSeededRepo() (*InMemoryRepo, error) {
	repo := NewInMemoryRepo()

	for _, film := range seedFilms {
		if err := repo.AddFilm(film); err != nil {
			return nil, fmt.Errorf("seed films: %w", err)
		}
	}
	for _, seed := range seedCollections {
		if err := repo.AddCollection(seed.collection, seed.filmIDs...); err != nil {
			return nil, fmt.Errorf("seed collections: %w", err)
		}
	}

	return repo, nil
}
