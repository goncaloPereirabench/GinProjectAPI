package store

import (
	"context"

	"ginprojectapi/internal/domain"

	"github.com/google/uuid"
)

type Repositories struct {
	Users    UserRepository
	Products ProductRepository
	Carts    CartRepository
}

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
}

type ProductRepository interface {
	Create(ctx context.Context, product domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]domain.Product, error)
	Update(ctx context.Context, product domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProductFilter struct {
	Query           string
	IncludeInactive bool
	Limit           int
	Offset          int
}

type CartRepository interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error)
	SetItem(ctx context.Context, userID uuid.UUID, product domain.Product, quantity int) error
	RemoveItem(ctx context.Context, userID uuid.UUID, productID uuid.UUID) error
	Clear(ctx context.Context, userID uuid.UUID) error
}
