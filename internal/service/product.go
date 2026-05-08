package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/store"

	"github.com/google/uuid"
)

type ProductService struct {
	products store.ProductRepository
}

type ProductInput struct {
	Name        string
	SKU         string
	Description string
	PriceCents  int64
	Stock       int
	Active      bool
}

func NewProductService(products store.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

func (s *ProductService) Create(ctx context.Context, input ProductInput) (domain.Product, error) {
	now := time.Now().UTC()
	product := domain.Product{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(input.Name),
		SKU:         strings.TrimSpace(strings.ToUpper(input.SKU)),
		Description: strings.TrimSpace(input.Description),
		PriceCents:  input.PriceCents,
		Stock:       input.Stock,
		Active:      input.Active,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if product.Name == "" || product.SKU == "" || product.PriceCents < 0 || product.Stock < 0 {
		return domain.Product{}, store.ErrInvalid
	}
	return product, s.products.Create(ctx, product)
}

func (s *ProductService) List(ctx context.Context, filter store.ProductFilter) ([]domain.Product, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.products.List(ctx, filter)
}

func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return s.products.GetByID(ctx, id)
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, input ProductInput) (domain.Product, error) {
	existing, err := s.products.GetByID(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.SKU = strings.TrimSpace(strings.ToUpper(input.SKU))
	existing.Description = strings.TrimSpace(input.Description)
	existing.PriceCents = input.PriceCents
	existing.Stock = input.Stock
	existing.Active = input.Active
	existing.UpdatedAt = time.Now().UTC()

	if existing.Name == "" || existing.SKU == "" || existing.PriceCents < 0 || existing.Stock < 0 {
		return domain.Product{}, store.ErrInvalid
	}

	if err := s.products.Update(ctx, existing); err != nil {
		return domain.Product{}, err
	}
	return existing, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.products.Delete(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return err
	}
	return err
}
