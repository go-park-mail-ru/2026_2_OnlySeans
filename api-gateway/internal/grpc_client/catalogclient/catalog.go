package catalogclient

import (
	"context"
	"fmt"
	"math"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type CatalogClient interface {
	ListFilms(ctx context.Context, genreSlug string, limit, offset int) (*pb.ListFilmsResponse, error)
	ListGenres(ctx context.Context) (*pb.ListGenresResponse, error)
	GetFilm(ctx context.Context, id int64) (*pb.GetFilmResponse, error)
	CreateFilm(ctx context.Context, input *pb.FilmInput) (*pb.CreateFilmResponse, error)
	UpdateFilm(ctx context.Context, id int64, input *pb.FilmInput) (*emptypb.Empty, error)
	DeleteFilm(ctx context.Context, id int64) (*emptypb.Empty, error)
	ListCollections(ctx context.Context, limit, offset int) (*pb.ListCollectionsResponse, error)
	GetCollection(ctx context.Context, id int64, limit, offset int) (*pb.GetCollectionResponse, error)
	SetFilmRating(ctx context.Context, user, film int64, score int) (*pb.SetFilmRatingResponse, error)
	GetFilmRating(ctx context.Context, id int64) (*pb.GetFilmRatingResponse, error)
	GetMyFilmRating(ctx context.Context, user, film int64) (*pb.GetMyFilmRatingResponse, error)
	DeleteFilmRating(ctx context.Context, user, film int64) (*emptypb.Empty, error)
	GetCollectionRatings(ctx context.Context, id int64, limit, offset int) (*pb.GetCollectionRatingsResponse, error)
	GetLibrary(ctx context.Context, user int64) (*pb.GetLibraryResponse, error)
	SetSaved(ctx context.Context, user int64, list string, film int64, saved bool) (*emptypb.Empty, error)
	GetPreferences(ctx context.Context, user int64) (*pb.GetPreferencesResponse, error)
	UploadPoster(ctx context.Context, user, film int64, data []byte) (*pb.UploadPosterResponse, error)
	GetPosterURL(ctx context.Context, id int64) (*pb.GetPosterURLResponse, error)
	DeletePoster(ctx context.Context, id int64) (*emptypb.Empty, error)
	SavePreference(ctx context.Context, request *pb.SavePreferenceRequest) (*emptypb.Empty, error)
	TopGenres(ctx context.Context, from, to time.Time, limit int) (*pb.TopGenresResponse, error)
	GenreAge(ctx context.Context, genreSlug string, from, to time.Time) (*pb.GenreAgeResponse, error)
	RecordFilmOpen(ctx context.Context, visitorPublicID string, accountID *int64, filmID int64) error
}

type catalogClient struct {
	client pb.CatalogServiceClient
}

func NewCatalogClient(client pb.CatalogServiceClient) CatalogClient {
	return &catalogClient{
		client: client,
	}
}

func pagination(limit, offset int) *pb.Pagination {
	value := int64(limit)

	return &pb.Pagination{
		Limit:  &value,
		Offset: int64(offset),
	}
}

func (c *catalogClient) ListFilms(ctx context.Context, genreSlug string, limit, offset int) (*pb.ListFilmsResponse, error) {
	result, err := c.client.ListFilms(ctx, &pb.ListFilmsRequest{
		Pagination: pagination(limit, offset),
		GenreSlug:  genreSlug,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) ListGenres(ctx context.Context) (*pb.ListGenresResponse, error) {
	result, err := c.client.ListGenres(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) TopGenres(ctx context.Context, from, to time.Time, limit int) (*pb.TopGenresResponse, error) {
	result, err := c.client.TopGenres(ctx, &pb.TopGenresRequest{From: timestamppb.New(from), To: timestamppb.New(to), Limit: int32(limit)})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GenreAge(ctx context.Context, genreSlug string, from, to time.Time) (*pb.GenreAgeResponse, error) {
	result, err := c.client.GenreAge(ctx, &pb.GenreAgeRequest{GenreSlug: genreSlug, From: timestamppb.New(from), To: timestamppb.New(to)})
	if err != nil {
		return nil, fromGRPCError(err)
	}
	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) RecordFilmOpen(ctx context.Context, visitorPublicID string, accountID *int64, filmID int64) error {
	_, err := c.client.RecordFilmOpen(ctx, &pb.RecordFilmOpenRequest{VisitorPublicId: visitorPublicID, AccountId: accountID, FilmId: filmID})
	if err != nil {
		return fromGRPCError(err)
	}

	return nil
}

func (c *catalogClient) GetFilm(ctx context.Context, id int64) (*pb.GetFilmResponse, error) {
	result, err := c.client.GetFilm(ctx, &pb.GetFilmRequest{
		FilmId: id,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) CreateFilm(ctx context.Context, input *pb.FilmInput) (*pb.CreateFilmResponse, error) {
	result, err := c.client.CreateFilm(ctx, &pb.CreateFilmRequest{
		Film: input,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) UpdateFilm(ctx context.Context, id int64, input *pb.FilmInput) (*emptypb.Empty, error) {
	result, err := c.client.UpdateFilm(ctx, &pb.UpdateFilmRequest{
		FilmId: id,
		Film:   input,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) DeleteFilm(ctx context.Context, id int64) (*emptypb.Empty, error) {
	result, err := c.client.DeleteFilm(ctx, &pb.DeleteFilmRequest{
		FilmId: id,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) ListCollections(ctx context.Context, limit, offset int) (*pb.ListCollectionsResponse, error) {
	result, err := c.client.ListCollections(ctx, &pb.ListCollectionsRequest{
		Pagination: pagination(limit, offset),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetCollection(ctx context.Context, id int64, limit, offset int) (*pb.GetCollectionResponse, error) {
	result, err := c.client.GetCollection(ctx, &pb.GetCollectionRequest{
		CollectionId: id,
		Pagination:   pagination(limit, offset),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) SetFilmRating(ctx context.Context, user, film int64, score int) (*pb.SetFilmRatingResponse, error) {
	if score < math.MinInt32 || score > math.MaxInt32 {
		return nil, fmt.Errorf("%w: must be from 1 to 10", ErrInvalidFilmRating)
	}

	result, err := c.client.SetFilmRating(ctx, &pb.SetFilmRatingRequest{
		UserId: user,
		FilmId: film,
		Score:  int32(score),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetFilmRating(ctx context.Context, id int64) (*pb.GetFilmRatingResponse, error) {
	result, err := c.client.GetFilmRating(ctx, &pb.GetFilmRatingRequest{
		FilmId: id,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetMyFilmRating(ctx context.Context, user, film int64) (*pb.GetMyFilmRatingResponse, error) {
	result, err := c.client.GetMyFilmRating(ctx, &pb.GetMyFilmRatingRequest{
		UserId: user,
		FilmId: film,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) DeleteFilmRating(ctx context.Context, user, film int64) (*emptypb.Empty, error) {
	result, err := c.client.DeleteFilmRating(ctx, &pb.DeleteFilmRatingRequest{
		UserId: user,
		FilmId: film,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetCollectionRatings(ctx context.Context, id int64, limit, offset int) (*pb.GetCollectionRatingsResponse, error) {
	result, err := c.client.GetCollectionRatings(ctx, &pb.GetCollectionRatingsRequest{
		CollectionId: id,
		Pagination:   pagination(limit, offset),
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetLibrary(ctx context.Context, user int64) (*pb.GetLibraryResponse, error) {
	result, err := c.client.GetLibrary(ctx, &pb.GetLibraryRequest{
		UserId: user,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) SetSaved(ctx context.Context, user int64, list string, film int64, saved bool) (*emptypb.Empty, error) {
	result, err := c.client.SetSaved(ctx, &pb.SetSavedRequest{
		UserId: user,
		List:   list,
		FilmId: film,
		Saved:  saved,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetPreferences(ctx context.Context, user int64) (*pb.GetPreferencesResponse, error) {
	result, err := c.client.GetPreferences(ctx, &pb.GetPreferencesRequest{
		UserId: user,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) UploadPoster(ctx context.Context, user, film int64, data []byte) (*pb.UploadPosterResponse, error) {
	result, err := c.client.UploadPoster(ctx, &pb.UploadPosterRequest{
		UserId:    user,
		FilmId:    film,
		ImageData: data,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) GetPosterURL(ctx context.Context, id int64) (*pb.GetPosterURLResponse, error) {
	result, err := c.client.GetPosterURL(ctx, &pb.GetPosterURLRequest{
		FilmId: id,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) DeletePoster(ctx context.Context, id int64) (*emptypb.Empty, error) {
	result, err := c.client.DeletePoster(ctx, &pb.DeletePosterRequest{
		FilmId: id,
	})
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}

func (c *catalogClient) SavePreference(ctx context.Context, request *pb.SavePreferenceRequest) (*emptypb.Empty, error) {
	result, err := c.client.SavePreference(ctx, request)
	if err != nil {
		return nil, fromGRPCError(err)
	}

	if result == nil {
		return nil, ErrInternal
	}

	return result, nil
}
