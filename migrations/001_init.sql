CREATE TABLE users (
    id UNIQUEIDENTIFIER NOT NULL PRIMARY KEY,
    email NVARCHAR(254) NOT NULL UNIQUE,
    password_hash NVARCHAR(255) NOT NULL,
    role NVARCHAR(40) NOT NULL,
    created_at DATETIME2 NOT NULL
);

CREATE TABLE products (
    id UNIQUEIDENTIFIER NOT NULL PRIMARY KEY,
    name NVARCHAR(120) NOT NULL,
    sku NVARCHAR(64) NOT NULL UNIQUE,
    description NVARCHAR(500) NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL,
    stock INT NOT NULL,
    active BIT NOT NULL DEFAULT 1,
    created_at DATETIME2 NOT NULL,
    updated_at DATETIME2 NOT NULL,
    CONSTRAINT ck_products_price_cents CHECK (price_cents >= 0),
    CONSTRAINT ck_products_stock CHECK (stock >= 0)
);

CREATE INDEX ix_products_active_name ON products (active, name);

CREATE TABLE cart_items (
    user_id UNIQUEIDENTIFIER NOT NULL,
    product_id UNIQUEIDENTIFIER NOT NULL,
    quantity INT NOT NULL,
    created_at DATETIME2 NOT NULL,
    updated_at DATETIME2 NOT NULL,
    CONSTRAINT pk_cart_items PRIMARY KEY (user_id, product_id),
    CONSTRAINT fk_cart_items_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cart_items_products FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT ck_cart_items_quantity CHECK (quantity > 0)
);
