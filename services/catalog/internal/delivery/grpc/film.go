package grpc

import (
	"context"
	"math"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) CreateFilm(ctx context.Context, req *pb.CreateFilmRequest) (*pb.CreateFilmResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	input, err := parseFilmInput(req.GetFilm())
	if err != nil {
		return nil, toGRPCError(err)
	}

	id, err := h.films.CreateFilm(ctx, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.CreateFilmResponse{
		FilmId: int64(id),
	}, nil
}

func (h *CatalogHandler) GetFilm(ctx context.Context, req *pb.GetFilmRequest) (*pb.GetFilmResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	film, err := h.films.GetFilmByID(ctx, domain.FilmID(req.GetFilmId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetFilmResponse{
		Film: toProtoFilm(film),
	}, nil
}

func (h *CatalogHandler) ListFilms(ctx context.Context, req *pb.ListFilmsRequest) (*pb.ListFilmsResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	limit, offset, err := parsePagination(req.GetPagination())
	if err != nil {
		return nil, toGRPCError(err)
	}

	films, total, err := h.films.ListFilms(ctx, limit, offset)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ListFilmsResponse{
		Films: toProtoFilms(films),
		Total: int64(total),
	}, nil
}

func (h *CatalogHandler) UpdateFilm(ctx context.Context, req *pb.UpdateFilmRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	input, err := parseFilmInput(req.GetFilm())
	if err != nil {
		return nil, toGRPCError(err)
	}

	if err := h.films.UpdateFilm(ctx, domain.FilmID(req.GetFilmId()), input); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *CatalogHandler) DeleteFilm(ctx context.Context, req *pb.DeleteFilmRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.films.DeleteFilm(ctx, domain.FilmID(req.GetFilmId())); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func parseFilmInput(input *pb.FilmInput) (domain.CreateFilmInput, error) {
	if input == nil || input.AgeLimit == nil {
		return domain.CreateFilmInput{}, domain.ErrInvalidFilmInput
	}

	year, err := filmInt16(input.GetReleaseYear())
	if err != nil {
		return domain.CreateFilmInput{}, err
	}

	age, err := filmInt16(input.GetAgeLimit())
	if err != nil {
		return domain.CreateFilmInput{}, err
	}

	var duration *int16

	if input.DurationMin != nil {
		value, err := filmInt16(input.GetDurationMin())
		if err != nil {
			return domain.CreateFilmInput{}, err
		}

		duration = &value
	}

	return domain.CreateFilmInput{
		Title:         input.GetTitle(),
		OriginalTitle: input.OriginalTitle,
		FilmType:      domain.FilmType(input.GetFilmType()),
		ReleaseYear:   year,
		DurationMin:   duration,
		AgeLimit:      age,
		Description:   input.Description,
	}, nil
}

func filmInt16(value int32) (int16, error) {
	if value < math.MinInt16 || value > math.MaxInt16 {
		return 0, domain.ErrInvalidFilmInput
	}

	return int16(value), nil
}

func toProtoFilm(film domain.Film) *pb.Film {
	genres := make([]*pb.Genre, len(film.Genres))
	for i, genre := range film.Genres {
		genres[i] = &pb.Genre{
			Id:   int64(genre.ID),
			Name: genre.Name,
			Slug: genre.Slug,
		}
	}

	var duration *int32

	if film.DurationMin != nil {
		value := int32(*film.DurationMin)
		duration = &value
	}

	return &pb.Film{
		Id:            int64(film.ID),
		Title:         film.Title,
		OriginalTitle: film.OriginalTitle,
		FilmType:      string(film.FilmType),
		ReleaseYear:   int32(film.ReleaseYear),
		DurationMin:   duration,
		AgeLimit:      int32(film.AgeLimit),
		Description:   film.Description,
		PosterUrl:     film.PosterURL,
		TrailerUrl:    film.TrailerURL,
		Genres:        genres,
	}
}

func toProtoFilms(films []domain.Film) []*pb.Film {
	result := make([]*pb.Film, len(films))
	for i, film := range films {
		result[i] = toProtoFilm(film)
	}

	return result
}
