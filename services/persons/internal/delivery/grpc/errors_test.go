package grpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},
		{"missing person", domain.ErrPersonNotFound, codes.NotFound},
		{"wrapped missing person", fmt.Errorf("get: %w", domain.ErrPersonNotFound), codes.NotFound},
		{"invalid ID", domain.ErrInvalidPersonID, codes.InvalidArgument},
		{"invalid first name", domain.ErrInvalidFirstName, codes.InvalidArgument},
		{"invalid last name", domain.ErrInvalidLastName, codes.InvalidArgument},
		{"invalid pagination", domain.ErrInvalidPagination, codes.InvalidArgument},
		{"wrapped invalid date", fmt.Errorf("validate: %w", domain.ErrInvalidBirthDate), codes.InvalidArgument},
		{"database failure", errors.New("private database details"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toGRPCError(tt.err)
			if got := status.Code(err); got != tt.code {
				t.Fatalf("code = %v, want %v", got, tt.code)
			}
			if tt.code == codes.Internal && status.Convert(err).Message() != "internal server error" {
				t.Fatal("internal error exposes implementation details")
			}
		})
	}
}
