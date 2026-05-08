package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type Product struct {
	ID          uuid.UUID
	Name        string
	SKU         string
	Description string
	PriceCents  int64
	Stock       int
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CartItem struct {
	ProductID   uuid.UUID
	ProductName string
	Quantity    int
	UnitPrice   int64
	LineTotal   int64
}

type Cart struct {
	UserID     uuid.UUID
	Items      []CartItem
	TotalCents int64
	UpdatedAt  time.Time
}
