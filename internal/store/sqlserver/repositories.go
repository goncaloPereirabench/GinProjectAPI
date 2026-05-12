package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/store"

	"github.com/google/uuid"
	mssql "github.com/microsoft/go-mssqldb"
)

type repositories struct {
	db *sql.DB
}

func NewRepositories(db *sql.DB) store.Repositories {
	repos := &repositories{db: db}
	return store.Repositories{
		Users:    (*userRepository)(repos),
		Products: (*productRepository)(repos),
		Carts:    (*cartRepository)(repos),
	}
}

type userRepository repositories

func (r *userRepository) Create(ctx context.Context, user domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (@p1, @p2, @p3, @p4, @p5)
	`, toMSSQLUUID(user.ID), user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return store.ErrConflict
		}
		return err
	}
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = @p1
	`, email))
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE id = @p1
	`, toMSSQLUUID(id)))
}

func (r *userRepository) scanUser(row *sql.Row) (domain.User, error) {
	var user domain.User
	var id mssql.UniqueIdentifier
	err := row.Scan(&id, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, store.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}

	user.ID = fromMSSQLUUID(id)
	return user, nil
}

type productRepository repositories

func (r *productRepository) Create(ctx context.Context, product domain.Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products (id, name, sku, description, price_cents, stock, active, created_at, updated_at)
		VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9)
	`, toMSSQLUUID(product.ID), product.Name, product.SKU, product.Description, product.PriceCents, product.Stock, product.Active, product.CreatedAt, product.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return store.ErrConflict
		}
		return err
	}
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	var product domain.Product
	var productID mssql.UniqueIdentifier
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, sku, description, price_cents, stock, active, created_at, updated_at
		FROM products
		WHERE id = @p1
	`, toMSSQLUUID(id)).Scan(
		&productID,
		&product.Name,
		&product.SKU,
		&product.Description,
		&product.PriceCents,
		&product.Stock,
		&product.Active,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Product{}, store.ErrNotFound
	}
	if err != nil {
		return domain.Product{}, err
	}

	product.ID = fromMSSQLUUID(productID)
	return product, nil
}

func (r *productRepository) List(ctx context.Context, filter store.ProductFilter) ([]domain.Product, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, sku, description, price_cents, stock, active, created_at, updated_at
		FROM products
		WHERE (@p1 = 1 OR active = 1)
		  AND (@p2 = '' OR LOWER(name) LIKE '%' + LOWER(@p2) + '%' OR LOWER(sku) LIKE '%' + LOWER(@p2) + '%')
		ORDER BY name
		OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY
	`, filter.IncludeInactive, filter.Query, filter.Offset, filter.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		var product domain.Product
		var productID mssql.UniqueIdentifier
		if err := rows.Scan(
			&productID,
			&product.Name,
			&product.SKU,
			&product.Description,
			&product.PriceCents,
			&product.Stock,
			&product.Active,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}
		product.ID = fromMSSQLUUID(productID)
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *productRepository) Update(ctx context.Context, product domain.Product) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE products
		SET name = @p2,
		    sku = @p3,
		    description = @p4,
		    price_cents = @p5,
		    stock = @p6,
		    active = @p7,
		    updated_at = @p8
		WHERE id = @p1
	`, toMSSQLUUID(product.ID), product.Name, product.SKU, product.Description, product.PriceCents, product.Stock, product.Active, product.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return store.ErrConflict
		}
		return err
	}
	return rowsAffectedError(result)
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM products
		WHERE id = @p1
	`, toMSSQLUUID(id))
	if err != nil {
		return err
	}
	return rowsAffectedError(result)
}

type cartRepository repositories

func (r *cartRepository) Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ci.product_id, p.name, ci.quantity, p.price_cents, ci.quantity * p.price_cents
		FROM cart_items ci
		INNER JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = @p1
		ORDER BY p.name
	`, toMSSQLUUID(userID))
	if err != nil {
		return domain.Cart{}, err
	}
	defer rows.Close()

	cart := domain.Cart{
		UserID: userID,
		Items:  make([]domain.CartItem, 0),
	}
	for rows.Next() {
		var item domain.CartItem
		var productID mssql.UniqueIdentifier
		if err := rows.Scan(&productID, &item.ProductName, &item.Quantity, &item.UnitPrice, &item.LineTotal); err != nil {
			return domain.Cart{}, err
		}
		item.ProductID = fromMSSQLUUID(productID)
		cart.Items = append(cart.Items, item)
		cart.TotalCents += item.LineTotal
	}
	return cart, rows.Err()
}

func (r *cartRepository) SetItem(ctx context.Context, userID uuid.UUID, product domain.Product, quantity int) error {
	_, err := r.db.ExecContext(ctx, `
		MERGE cart_items AS target
		USING (SELECT @p1 AS user_id, @p2 AS product_id) AS source
		ON target.user_id = source.user_id AND target.product_id = source.product_id
		WHEN MATCHED THEN
			UPDATE SET quantity = @p3, updated_at = SYSUTCDATETIME()
		WHEN NOT MATCHED THEN
			INSERT (user_id, product_id, quantity, created_at, updated_at)
			VALUES (source.user_id, source.product_id, @p3, SYSUTCDATETIME(), SYSUTCDATETIME());
	`, toMSSQLUUID(userID), toMSSQLUUID(product.ID), quantity)
	return err
}

func (r *cartRepository) RemoveItem(ctx context.Context, userID uuid.UUID, productID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM cart_items
		WHERE user_id = @p1 AND product_id = @p2
	`, toMSSQLUUID(userID), toMSSQLUUID(productID))
	return err
}

func (r *cartRepository) Clear(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM cart_items
		WHERE user_id = @p1
	`, toMSSQLUUID(userID))
	return err
}

func toMSSQLUUID(id uuid.UUID) mssql.UniqueIdentifier {
	return mssql.UniqueIdentifier(id)
}

func fromMSSQLUUID(id mssql.UniqueIdentifier) uuid.UUID {
	return uuid.UUID(id)
}

func rowsAffectedError(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return store.ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "2627") || strings.Contains(err.Error(), "2601"))
}
