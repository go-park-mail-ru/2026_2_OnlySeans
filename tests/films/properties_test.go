package films_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
)

func TestInMemoryService_PagesCoverAllFilmsOnce(t *testing.T) {
	const filmCount = 17

	service := newTestService(t, filmCount)
	ctx := context.Background()

	want := make([]entities.FilmID, 0, filmCount)
	for id := entities.FilmID(1); id <= filmCount; id++ {
		want = append(want, id)
	}

	for limit := 1; limit <= filmCount+3; limit++ {
		var got []entities.FilmID
		for offset := 0; ; offset += limit {
			page, total, err := service.ListFilms(ctx, limit, offset)
			if err != nil {
				t.Fatalf("limit %d, offset %d: %v", limit, offset, err)
			}
			if total != filmCount {
				t.Fatalf("limit %d, offset %d: total = %d, want %d", limit, offset, total, filmCount)
			}
			if len(page) == 0 {
				break
			}
			if len(page) > limit {
				t.Fatalf("limit %d, offset %d: page has %d films", limit, offset, len(page))
			}
			got = append(got, filmIDs(page)...)
		}

		if !slices.Equal(got, want) {
			t.Errorf("limit %d: pages gave %v, want %v", limit, got, want)
		}
	}
}

func TestInMemoryService_ConcurrentReadsAndWrites(t *testing.T) {
	const filmCount = 200

	service := films.NewInMemoryService()
	ctx := context.Background()

	var wg sync.WaitGroup
	for id := entities.FilmID(1); id <= filmCount; id++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := service.AddFilm(entities.Film{ID: id, Genres: []entities.Genre{genreDrama}}); err != nil {
				t.Errorf("AddFilm(%d): %v", id, err)
			}
		}()
		go func() {
			defer wg.Done()
			page, total, err := service.ListFilms(ctx, filmCount, 0)
			if err != nil {
				t.Errorf("ListFilms: %v", err)
			}
			if len(page) != total {
				t.Errorf("page has %d films, total = %d", len(page), total)
			}
			if !slices.IsSorted(filmIDs(page)) {
				t.Errorf("films are not sorted by id: %v", filmIDs(page))
			}
		}()
	}
	wg.Wait()

	page, total, err := service.ListFilms(ctx, filmCount, 0)
	if err != nil || total != filmCount || len(page) != filmCount {
		t.Fatalf("ListFilms = %d films, total %d, err %v", len(page), total, err)
	}
	if ids := filmIDs(page); !slices.IsSorted(ids) || ids[0] != 1 || ids[filmCount-1] != filmCount {
		t.Errorf("ids = %v", ids)
	}
}

func TestHandler_FilmJSONContract(t *testing.T) {
	rec := serve(newSeededRouter(t), http.MethodGet, "/api/films?limit=100")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}

	body := decode[struct {
		Films []map[string]json.RawMessage `json:"films"`
	}](t, rec)
	if len(body.Films) != seedFilmCount {
		t.Fatalf("got %d films, want %d", len(body.Films), seedFilmCount)
	}

	required := []string{"id", "title", "film_type", "production_year", "age_limit", "genres"}
	for i, film := range body.Films {
		for _, key := range required {
			if _, ok := film[key]; !ok {
				t.Errorf("film %d: no field %q", i, key)
			}
		}
		for key, value := range film {
			if string(value) == "null" {
				t.Errorf("film %d: field %q is null", i, key)
			}
		}
		if !strings.HasPrefix(string(film["genres"]), "[") {
			t.Errorf("film %d: genres = %s, want array", i, film["genres"])
		}
	}
}

func FuzzHandler_PathID(f *testing.F) {
	for _, id := range []string{
		"1",
		"4",
		"12",
		"0",
		"-1",
		"+1",
		"01",
		" 1",
		"1.0",
		"1e0",
		"0x1",
		"family",
		"",
		"../../etc/passwd",
		"1/extra",
		"один",
		"1%00",
		"1 OR 1=1",
		"9223372036854775808",
		strings.Repeat("9", 200),
	} {
		f.Add(id)
	}

	service, err := films.NewSeededService()
	if err != nil {
		f.Fatalf("NewSeededService: %v", err)
	}
	router := newTestRouter(service)

	f.Fuzz(func(t *testing.T, id string) {
		parsed, parseErr := strconv.ParseInt(id, 10, 64)
		valid := parseErr == nil && parsed > 0

		for prefix, maxID := range map[string]int64{"/api/films/": seedFilmCount, "/api/collections/": int64(len(seedCollectionFilms))} {
			rec := serve(router, http.MethodGet, prefix+url.PathEscape(id))

			if rec.Code >= http.StatusInternalServerError {
				t.Fatalf("%s%q: status = %d (body: %s)", prefix, id, rec.Code, rec.Body)
			}
			if rec.Code == http.StatusOK && !(valid && parsed <= maxID) {
				t.Fatalf("%s%q was accepted", prefix, id)
			}
		}
	})
}

func TestHandler_CollectionPagesCoverAllFilmsOnce(t *testing.T) {
	router := newSeededRouter(t)

	for i, seed := range seedCollectionFilms {
		for limit := 1; limit <= len(seed.filmIDs)+1; limit++ {
			var got []entities.FilmID
			for offset := 0; ; offset += limit {
				target := fmt.Sprintf("/api/collections/%d?limit=%d&offset=%d", i+1, limit, offset)
				rec := serve(router, http.MethodGet, target)
				if rec.Code != http.StatusOK {
					t.Fatalf("%s: status = %d", target, rec.Code)
				}

				page := decode[collectionPage](t, rec)
				if page.Total != len(seed.filmIDs) || page.Limit != limit || page.Offset != offset {
					t.Fatalf("%s: total %d, limit %d, offset %d", target, page.Total, page.Limit, page.Offset)
				}
				if len(page.Collection.Films) == 0 {
					break
				}
				got = append(got, filmIDs(page.Collection.Films)...)
			}

			if !slices.Equal(got, seed.filmIDs) {
				t.Errorf("collection %q, limit %d: pages gave %v, want %v", seed.slug, limit, got, seed.filmIDs)
			}
		}
	}
}

func FuzzHandler_ListFilmsParams(f *testing.F) {
	for _, params := range [][2]string{
		{"20", "0"},
		{"", ""},
		{"abc", "-1"},
		{"0", "0"},
		{"101", "5"},
		{"1", "9223372036854775807"},
		{"99999999999999999999999", "1"},
		{"1e2", "0x10"},
		{" 5", "5 "},
		{"+5", "+0"},
	} {
		f.Add(params[0], params[1])
	}

	service, err := films.NewSeededService()
	if err != nil {
		f.Fatalf("NewSeededService: %v", err)
	}
	router := newTestRouter(service)

	f.Fuzz(func(t *testing.T, limit, offset string) {
		query := url.Values{"limit": {limit}, "offset": {offset}}
		rec := serve(router, http.MethodGet, "/api/films?"+query.Encode())

		switch rec.Code {
		case http.StatusBadRequest:
		case http.StatusOK:
			page := decode[filmsPage](t, rec)
			if page.Limit < 1 || page.Limit > 100 || page.Offset < 0 {
				t.Fatalf("limit=%q offset=%q accepted as %d, %d", limit, offset, page.Limit, page.Offset)
			}
			if len(page.Films) > page.Limit {
				t.Fatalf("limit=%q: got %d films", limit, len(page.Films))
			}
		default:
			t.Fatalf("limit=%q offset=%q: status = %d (body: %s)", limit, offset, rec.Code, rec.Body)
		}
	})
}
