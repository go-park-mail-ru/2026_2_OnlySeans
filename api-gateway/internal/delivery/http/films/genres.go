package films

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
)

type genreResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type genresResponse struct {
	Genres []genreResponse `json:"genres"`
}

func (h *Handler) ListGenres(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.ListGenres(r.Context())
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	genres := make([]genreResponse, len(result.GetGenres()))
	for i, genre := range result.GetGenres() {
		genres[i] = genreResponse{ID: genre.GetId(), Name: genre.GetName(), Slug: genre.GetSlug()}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, genresResponse{Genres: genres}, h.appLogger)
}
