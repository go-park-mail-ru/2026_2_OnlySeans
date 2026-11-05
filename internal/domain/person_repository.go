package domain

import "context"

type PersonRepository interface {
	Create(ctx context.Context, input PersonInput) (PersonID, error)
	GetByID(ctx context.Context, id PersonID) (*Person, error)
	List(ctx context.Context, limit, offset int) ([]Person, int, error)
	Update(ctx context.Context, id PersonID, input PersonInput) (*Person, error)
	Delete(ctx context.Context, id PersonID) error
}
