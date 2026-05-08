package service

import (
	"context"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/store"

	"github.com/google/uuid"
)

type CartService struct {
	products store.ProductRepository
	carts    store.CartRepository
}

func NewCartService(products store.ProductRepository, carts store.CartRepository) *CartService {
	return &CartService{products: products, carts: carts}
}

func (s *CartService) Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error) {
	return s.carts.Get(ctx, userID)
}

func (s *CartService) SetItem(ctx context.Context, userID, productID uuid.UUID, quantity int) (domain.Cart, error) {
	if quantity < 0 {
		return domain.Cart{}, store.ErrInvalid
	}

	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return domain.Cart{}, err
	}
	if !product.Active || product.Stock < quantity {
		return domain.Cart{}, store.ErrInvalid
	}

	if quantity == 0 {
		if err := s.carts.RemoveItem(ctx, userID, productID); err != nil {
			return domain.Cart{}, err
		}
		return s.carts.Get(ctx, userID)
	}

	if err := s.carts.SetItem(ctx, userID, product, quantity); err != nil {
		return domain.Cart{}, err
	}
	return s.carts.Get(ctx, userID)
}

func (s *CartService) RemoveItem(ctx context.Context, userID, productID uuid.UUID) (domain.Cart, error) {
	if err := s.carts.RemoveItem(ctx, userID, productID); err != nil {
		return domain.Cart{}, err
	}
	return s.carts.Get(ctx, userID)
}

func (s *CartService) Clear(ctx context.Context, userID uuid.UUID) error {
	return s.carts.Clear(ctx, userID)
}
