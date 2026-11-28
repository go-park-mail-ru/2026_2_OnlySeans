package grpc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	cataloggrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/releaselimits"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func TestMaximumReleasePageOmitsLongDescriptions(t *testing.T) {
	description := strings.Repeat("я", 4700)

	releases := make([]domain.Release, releaselimits.MaxLimit)
	for i := range releases {
		releases[i].Film = sampleFilm()
		releases[i].Film.Description = &description
	}

	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
	repo.EXPECT().List(gomock.Any(), gomock.Any(), int64(0)).Return(releases, int64(len(releases)), nil)
	handler := cataloggrpc.NewReleaseHandler(usecase.NewReleaseUseCase(repo, nil))
	result, err := handler.ListReleases(t.Context(), &pb.ListReleasesRequest{From: "2026-12-01", To: "2027-01-01", Limit: releaselimits.MaxLimit})
	require.NoError(t, err)
	require.Len(t, result.Items, releaselimits.MaxLimit)
	require.Less(t, proto.Size(result), 4*1024*1024)
	require.Nil(t, result.Items[0].Film.Description)
	_, err = handler.ListReleases(t.Context(), &pb.ListReleasesRequest{From: "2200-01-01", To: "2200-01-03"})
	require.Error(t, err)
}
