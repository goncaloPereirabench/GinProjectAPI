package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/store"

	"github.com/google/uuid"
)

type repositories struct {
	users    *userRepository
	products *productRepository
	carts    *cartRepository
}

func NewRepositories() store.Repositories {
	repos := &repositories{
		users: &userRepository{
			byID:    make(map[uuid.UUID]domain.User),
			byEmail: make(map[string]uuid.UUID),
		},
		products: &productRepository{
			byID:  make(map[uuid.UUID]domain.Product),
			bySKU: make(map[string]uuid.UUID),
		},
		carts: &cartRepository{
			items: make(map[uuid.UUID]map[uuid.UUID]domain.CartItem),
		},
	}

	return store.Repositories{
		Users:    repos.users,
		Products: repos.products,
		Carts:    repos.carts,
	}
}

type userRepository struct {
	mu      sync.RWMutex
	byID    map[uuid.UUID]domain.User
	byEmail map[string]uuid.UUID
}

func (r *userRepository) Create(_ context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	email := strings.ToLower(user.Email)
	if _, exists := r.byEmail[email]; exists {
		return store.ErrConflict
	}

	r.byID[user.ID] = user
	r.byEmail[email] = user.ID
	return nil
}

func (r *userRepository) GetByEmail(_ context.Context, email string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.byEmail[strings.ToLower(email)]
	if !exists {
		return domain.User{}, store.ErrNotFound
	}
	return r.byID[id], nil
}

func (r *userRepository) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.byID[id]
	if !exists {
		return domain.User{}, store.ErrNotFound
	}
	return user, nil
}

type productRepository struct {
	mu    sync.RWMutex
	byID  map[uuid.UUID]domain.Product
	bySKU map[string]uuid.UUID
}

func (r *productRepository) Create(_ context.Context, product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sku := strings.ToUpper(product.SKU)
	if _, exists := r.bySKU[sku]; exists {
		return store.ErrConflict
	}

	r.byID[product.ID] = product
	r.bySKU[sku] = product.ID
	return nil
}

func (r *productRepository) GetByID(_ context.Context, id uuid.UUID) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, exists := r.byID[id]
	if !exists {
		return domain.Product{}, store.ErrNotFound
	}
	return product, nil
}

func (r *productRepository) List(_ context.Context, filter store.ProductFilter) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(filter.Query))
	products := make([]domain.Product, 0, len(r.byID))
	for _, product := range r.byID {
		if !filter.IncludeInactive && !product.Active {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(product.Name), query) && !strings.Contains(strings.ToLower(product.SKU), query) {
			continue
		}
		products = append(products, product)
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].Name < products[j].Name
	})

	if filter.Offset >= len(products) {
		return []domain.Product{}, nil
	}

	end := filter.Offset + filter.Limit
	if end > len(products) {
		end = len(products)
	}
	return products[filter.Offset:end], nil
}

func (r *productRepository) Update(_ context.Context, product domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.byID[product.ID]
	if !exists {
		return store.ErrNotFound
	}

	newSKU := strings.ToUpper(product.SKU)
	if existingID, exists := r.bySKU[newSKU]; exists && existingID != product.ID {
		return store.ErrConflict
	}

	delete(r.bySKU, strings.ToUpper(current.SKU))
	r.byID[product.ID] = product
	r.bySKU[newSKU] = product.ID
	return nil
}

func (r *productRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	product, exists := r.byID[id]
	if !exists {
		return store.ErrNotFound
	}

	delete(r.byID, id)
	delete(r.bySKU, strings.ToUpper(product.SKU))
	return nil
}

type cartRepository struct {
	mu    sync.RWMutex
	items map[uuid.UUID]map[uuid.UUID]domain.CartItem
}

func (r *cartRepository) Get(_ context.Context, userID uuid.UUID) (domain.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userItems := r.items[userID]
	cart := domain.Cart{
		UserID:    userID,
		Items:     make([]domain.CartItem, 0, len(userItems)),
		UpdatedAt: time.Now().UTC(),
	}

	for _, item := range userItems {
		cart.Items = append(cart.Items, item)
		cart.TotalCents += item.LineTotal
	}

	sort.Slice(cart.Items, func(i, j int) bool {
		return cart.Items[i].ProductName < cart.Items[j].ProductName
	})

	return cart, nil
}

func (r *cartRepository) SetItem(_ context.Context, userID uuid.UUID, product domain.Product, quantity int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.items[userID] == nil {
		r.items[userID] = make(map[uuid.UUID]domain.CartItem)
	}

	r.items[userID][product.ID] = domain.CartItem{
		ProductID:   product.ID,
		ProductName: product.Name,
		Quantity:    quantity,
		UnitPrice:   product.PriceCents,
		LineTotal:   product.PriceCents * int64(quantity),
	}
	return nil
}

func (r *cartRepository) RemoveItem(_ context.Context, userID uuid.UUID, productID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.items[userID] == nil {
		return nil
	}
	delete(r.items[userID], productID)
	return nil
}

func (r *cartRepository) Clear(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.items, userID)
	return nil
}
