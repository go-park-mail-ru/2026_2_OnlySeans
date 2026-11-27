package grpc

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type CatalogHandler struct {
	pb.UnimplementedCatalogServiceServer
	films       usecase.FilmUseCase
	collections usecase.CollectionUseCase
	ratings     usecase.RatingUseCase
	library     usecase.LibraryUseCase
	preferences usecase.PreferencesUseCase
	posters     usecase.PosterUseCase
	analytics   usecase.AnalyticsUseCase
}

func NewCatalogHandler(films usecase.FilmUseCase, collections usecase.CollectionUseCase, ratings usecase.RatingUseCase, library usecase.LibraryUseCase, preferences usecase.PreferencesUseCase, posters usecase.PosterUseCase, analytics ...usecase.AnalyticsUseCase) *CatalogHandler {
	handler := &CatalogHandler{
		films:       films,
		collections: collections,
		ratings:     ratings,
		library:     library,
		preferences: preferences,
		posters:     posters,
	}
	if len(analytics) > 0 {
		handler.analytics = analytics[0]
	}

	return handler
}
