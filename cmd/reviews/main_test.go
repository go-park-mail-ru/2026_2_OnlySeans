package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	reviewsgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestServeListReviews(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)

	filmID := domain.FilmID(42)
	films.EXPECT().GetFilmByID(gomock.Any(), filmID).Return(domain.Film{}, nil)
	repo.EXPECT().ListByFilm(gomock.Any(), filmID, 20, 1).Return([]domain.Review{{ID: 7, FilmID: filmID, Title: "Отзыв"}}, 2, nil)

	listener := startReviewsServer(t, repo, films, "")
	client := reviewsClient(t, listener)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// если клиент не прислал Limit, сервис использует 20
	response, err := client.ListReviews(ctx, &reviewspb.ListReviewsRequest{FilmId: 42, Offset: 1})
	require.NoError(t, err)
	require.EqualValues(t, 2, response.GetTotal())
	require.Len(t, response.GetReviews(), 1)
	require.EqualValues(t, 7, response.GetReviews()[0].GetId())
	require.EqualValues(t, 42, response.GetReviews()[0].GetFilmId())
	require.Equal(t, "Отзыв", response.GetReviews()[0].GetTitle())

	_, err = client.ListReviews(ctx, &reviewspb.ListReviewsRequest{FilmId: -1})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestHTTPReviewsThroughGRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockReviewRepository(ctrl)
	films := mocks.NewMockFilmRepository(ctrl)

	const token = "test-reviews-token"
	const filmID, authorID, reviewID = domain.FilmID(42), domain.UserID(5), domain.ReviewID(7)
	const content = "Текст отзыва с достаточным количеством символов для проверки."
	original := domain.Review{ID: reviewID, FilmID: filmID, AuthorID: authorID, Title: "Обзор", Content: content, Version: 1}
	updated := original
	updated.Title, updated.Version, updated.ContainsSpoilers = "Новый обзор", 2, true
	edit := domain.ReviewUpdate{
		Input:   domain.ReviewInput{Title: updated.Title, Content: updated.Content, ContainsSpoilers: true},
		Version: 1,
	}

	// запрос проходит через HTTP, gRPC и настоящие юзкейсы до подменённых репозиториев
	films.EXPECT().GetFilmByID(gomock.Any(), filmID).Return(domain.Film{}, nil).Times(2)
	repo.EXPECT().Create(gomock.Cond(func(ctx context.Context) bool {
		return logger.RequestID(ctx) == "review-request"
	}), authorID, filmID, domain.ReviewInput{Title: original.Title, Content: content}).Return(&original, nil)
	repo.EXPECT().ListByFilm(gomock.Any(), filmID, 20, 0).Return([]domain.Review{original}, 1, nil)
	repo.EXPECT().GetByID(gomock.Any(), reviewID).Return(&original, nil).Times(4)
	gomock.InOrder(
		repo.EXPECT().Update(gomock.Any(), reviewID, edit).Return(&updated, nil),
		repo.EXPECT().Update(gomock.Any(), reviewID, edit).Return(nil, domain.ErrVersionConflict),
	)
	repo.EXPECT().Delete(gomock.Any(), reviewID, authorID).Return(nil)

	listener := startReviewsServer(t, repo, films, token)
	unauthenticated := reviewsClient(t, listener)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// переданного user_id недостаточно для вызова в обход HTTP-бэкенда
	_, err := unauthenticated.DeleteReview(ctx, &reviewspb.DeleteReviewRequest{ReviewId: 7, UserId: 5})
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	client := reviewsClient(t, listener, grpc.WithUnaryInterceptor(reviewsgrpc.AuthorizeRequest(token)))
	handler, err := reviewsdelivery.NewHandlerWithOperations(reviewsgrpc.ClientOperations(client), testutil.DiscardLogger())
	require.NoError(t, err)

	router := mux.NewRouter()
	router.HandleFunc("/films/{id}/reviews", handler.ListFilmReviews).Methods(http.MethodGet)
	router.HandleFunc("/films/{id}/reviews", handler.CreateReview).Methods(http.MethodPost)
	router.HandleFunc("/reviews/{id}", handler.UpdateReview).Methods(http.MethodPut)
	router.HandleFunc("/reviews/{id}", handler.DeleteReview).Methods(http.MethodDelete)

	call := func(method, path, body string, userID domain.UserID) *httptest.ResponseRecorder {
		ctx := authdelivery.ContextWithUserID(logger.ContextWithRequestID(ctx, "review-request"), userID)
		req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	createBody := `{"title":"Обзор","content":"` + content + `"}`
	response := call(http.MethodPost, "/films/42/reviews", createBody, authorID)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.JSONEq(t, `{"id":7}`, response.Body.String())

	response = call(http.MethodGet, "/films/42/reviews", "", authorID)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"total":1`)

	editBody := `{"title":"Новый обзор","content":"` + content + `","contains_spoilers":true,"version":1}`
	response = call(http.MethodPut, "/reviews/7", editBody, authorID)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"version":2`)

	response = call(http.MethodPut, "/reviews/7", editBody, authorID)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "review was changed by someone else")

	response = call(http.MethodDelete, "/reviews/7", "", 99)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	response = call(http.MethodDelete, "/reviews/7", "", authorID)
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.Empty(t, response.Body.String())
}

func startReviewsServer(t *testing.T, repo domain.ReviewRepository, films domain.FilmRepository, token string) *bufconn.Listener {
	t.Helper()
	operations, err := reviewsusecase.NewOperations(repo, films, testutil.DiscardLogger())
	require.NoError(t, err)
	handler, err := reviewsgrpc.NewHandler(operations)
	require.NoError(t, err)

	// bufconn передаёт реальные gRPC-сообщения через соединение в памяти
	listener := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = listener.Close() })
	go serve(listener, handler, token)
	return listener
}

func reviewsClient(t *testing.T, listener *bufconn.Listener, options ...grpc.DialOption) reviewspb.ReviewsServiceClient {
	t.Helper()
	options = append(options,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
	)
	conn, err := grpc.NewClient("passthrough:///reviews-test", options...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return reviewspb.NewReviewsServiceClient(conn)
}
