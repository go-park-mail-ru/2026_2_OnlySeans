package reviews_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type recordingPolicy struct {
	policy domain.ReviewPolicy
	calls  int
}

func (p *recordingPolicy) ReviewPolicy(domain.UserID) domain.ReviewPolicy {
	p.calls++
	return p.policy
}

func TestCreateReviewIsBlockedServerSideWhenFeatureDisabled(t *testing.T) {
	repo := mocks.NewMockReviewRepository(gomock.NewController(t))
	films := mocks.NewMockFilmRepository(gomock.NewController(t))
	policy := fixedReviewPolicy{policy: domain.ReviewPolicy{MaxLength: 500, Revision: 12}}
	uc, err := usecase.NewCreateReviewUseCase(repo, films, policy, newReviewsTestLogger())
	require.NoError(t, err)

	result, err := uc.Run(t.Context(), usecase.NewCreateReviewInput(7, 1, validInput()))
	require.Nil(t, result)
	require.ErrorIs(t, err, domain.ErrFeatureDisabled)
}

func TestCreateReviewUsesDynamicCodePointLimit(t *testing.T) {
	repo := mocks.NewMockReviewRepository(gomock.NewController(t))
	films := mocks.NewMockFilmRepository(gomock.NewController(t))
	policy := fixedReviewPolicy{policy: domain.ReviewPolicy{WriteEnabled: true, MaxLength: 50, Revision: 13}}
	uc, err := usecase.NewCreateReviewUseCase(repo, films, policy, newReviewsTestLogger())
	require.NoError(t, err)

	content := strings.Repeat("🙂", 51)
	result, err := uc.Run(t.Context(), usecase.NewCreateReviewInput(7, 1, domain.ReviewInput{Title: validTitle, Content: content}))
	require.Nil(t, result)
	var tooLong *domain.ReviewTooLongError
	require.ErrorAs(t, err, &tooLong)
	require.Equal(t, 50, tooLong.MaxLength)
	require.Equal(t, int64(13), tooLong.Revision)
}

func TestUpdateChecksOwnerBeforeOnlineConfPolicy(t *testing.T) {
	repo := mocks.NewMockReviewRepository(gomock.NewController(t))
	repo.EXPECT().GetByID(gomock.Any(), domain.ReviewID(1)).Return(&domain.Review{ID: 1, AuthorID: 7}, nil)
	policy := &recordingPolicy{policy: domain.ReviewPolicy{WriteEnabled: false, MaxLength: 50, Revision: 15}}
	uc, err := usecase.NewUpdateReviewUseCase(repo, policy, newReviewsTestLogger())
	require.NoError(t, err)

	result, err := uc.Run(context.Background(), usecase.NewUpdateReviewInput(8, 1, validInput(), 1))
	require.Nil(t, result)
	require.ErrorIs(t, err, domain.ErrForbidden)
	require.Zero(t, policy.calls, "foreign-resource checks must not reveal/evaluate policy")
}

func TestReviewHandlerReturnsStableBusinessCodes(t *testing.T) {
	disabled := fixedReviewPolicy{policy: domain.ReviewPolicy{MaxLength: 500, Revision: 21}}
	rec := serve(newTestRouterWithPolicy(t, testFilms(t, nil), disabled), "POST", filmReviewsPath(existingFilm), validBody, anna)
	require.Equal(t, 403, rec.Code)
	require.JSONEq(t, `{"error":"feature disabled","code":"feature_disabled"}`, rec.Body.String())

	limited := fixedReviewPolicy{policy: domain.ReviewPolicy{WriteEnabled: true, MaxLength: 50, Revision: 22}}
	body := reviewBody(validTitle, strings.Repeat("a", 51), false)
	rec = serve(newTestRouterWithPolicy(t, testFilms(t, nil), limited), "POST", filmReviewsPath(existingFilm), body, anna)
	require.Equal(t, 400, rec.Code)
	require.JSONEq(t, `{"error":"review content is too long","code":"review_too_long","details":{"max_length":50,"revision":22}}`, rec.Body.String())
}
