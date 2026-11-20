package search

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	searchusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/search"
)

type SearchUseCase interface {
	Run(
		ctx context.Context,
		input searchusecase.SearchSuggestionUsecaseInput,
	) (*searchusecase.SearchSuggestionUsecaseOutput, error)
}

type SearchHandler struct {
	search SearchUseCase
}

var _ StrictServerInterface = (*SearchHandler)(nil)

func NewSearchHandler(search SearchUseCase) (*SearchHandler, error) {
	if search == nil {
		return nil, errors.New("search suggestions use case is nil")
	}

	return &SearchHandler{search: search}, nil
}

func (h *SearchHandler) GetSearchSuggestions(ctx context.Context, request GetSearchSuggestionsRequestObject) (GetSearchSuggestionsResponseObject, error) {
	limit := 3
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	input := searchusecase.NewSearchSuggestionUsecaseInput(
		request.Params.Q,
		limit,
	)

	output, err := h.search.Run(ctx, *input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSearchQuery):
			return badSearchRequest(domain.ErrInvalidSearchQuery.Error()), nil

		case errors.Is(err, domain.ErrInvalidSearchLimit):
			return badSearchRequest(domain.ErrInvalidSearchLimit.Error()), nil

		default:
			return GetSearchSuggestions500JSONResponse{
				InternalErrorJSONResponse: InternalErrorJSONResponse{
					Error: "internal server error",
				},
			}, nil
		}
	}

	films := make([]SearchFilmSuggestion, 0, len(output.FilmSuggestions))
	for _, film := range output.FilmSuggestions {
		films = append(films, SearchFilmSuggestion{
			Id:            int64(film.ID),
			Title:         film.Title,
			OriginalTitle: film.OriginalTitle,
			Year:          int(film.ReleaseYear),
			FilmType:      SearchFilmSuggestionFilmType(film.FilmType),
			PosterUrl:     film.PosterURL,
		})
	}

	persons := make([]SearchPersonSuggestion, 0, len(output.PersonSuggestions))
	for _, person := range output.PersonSuggestions {
		persons = append(persons, SearchPersonSuggestion{
			Id:       int64(person.ID),
			Name:     person.Name,
			PhotoUrl: person.PhotoURL,
		})
	}

	return GetSearchSuggestions200JSONResponse{
		SearchSuggestionsJSONResponse: SearchSuggestionsJSONResponse{
			Films:   films,
			Persons: persons,
		},
	}, nil
}

func badSearchRequest(message string) GetSearchSuggestions400JSONResponse {
	return GetSearchSuggestions400JSONResponse{
		InvalidSearchRequestJSONResponse: InvalidSearchRequestJSONResponse{
			Error: message,
		},
	}
}
