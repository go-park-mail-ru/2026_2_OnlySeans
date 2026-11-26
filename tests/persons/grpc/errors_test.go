package grpc_test

import (
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	personsgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/tests/mocks"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
)

func TestToGRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "nil",
			err:  nil,
			code: codes.OK,
		},
		{
			name: "missing person",
			err:  domain.ErrPersonNotFound,
			code: codes.NotFound,
		},
		{
			name: "wrapped missing person",
			err:  fmt.Errorf("get: %w", domain.ErrPersonNotFound),
			code: codes.NotFound,
		},
		{
			name: "invalid ID",
			err:  domain.ErrInvalidPersonID,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid first name",
			err:  domain.ErrInvalidFirstName,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid last name",
			err:  domain.ErrInvalidLastName,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid pagination",
			err:  domain.ErrInvalidPagination,
			code: codes.InvalidArgument,
		},
		{
			name: "wrapped invalid date",
			err:  fmt.Errorf("validate: %w", domain.ErrInvalidBirthDate),
			code: codes.InvalidArgument,
		},
		{
			name: "database failure",
			err:  errors.New("private database details"),
			code: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := mocks.NewMockPersonUseCase(ctrl)
			catalog := mocks.NewMockCatalogUseCase(ctrl)

			uc.EXPECT().GetPersonByID(t.Context(), domain.PersonID(7)).Return(&domain.Person{}, tt.err)
			handler := personsgrpc.NewPersonHandler(uc, catalog)

			_, err := handler.GetByID(t.Context(), &pb.GetPersonRequest{
				PersonId: 7,
			})
			if got := status.Code(err); got != tt.code {
				t.Fatalf("code = %v, want %v", got, tt.code)
			}

			if tt.code == codes.Internal && status.Convert(err).Message() != "internal server error" {
				t.Fatal("internal error exposes implementation details")
			}
		})
	}
}
