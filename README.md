# GinProjectAPI

GinProjectAPI is a modern Go API built with the Gin framework around a grocery store domain. It includes JWT authentication, per-user request limiting, product inventory endpoints, a customer cart, Azure SQL Server storage, Docker packaging, and unit tests.

The project is intentionally layered so the business logic is not trapped inside HTTP handlers:

```text
cmd/api              application entry point and graceful shutdown
internal/config      environment-based configuration
internal/database    Azure SQL / SQL Server connection setup
internal/domain      core grocery-store models
internal/service     business use cases: auth, products, carts, JWT
internal/http        Gin router, handlers, middleware, JSON responses
internal/store       repository interfaces
internal/store/memory local dev and unit-test repositories
internal/store/sqlserver Azure SQL / SQL Server repositories
migrations           database schema
```

## What The API Does

The API models a small grocery store:

- Customers can register and log in.
- The API returns a signed JWT access token.
- Product listing and product details are public.
- Product changes and cart actions require a bearer token.
- Requests are throttled with a token-bucket limiter.
- Authenticated callers are limited by user token. Anonymous callers are limited by IP address.

For a horizontally scaled Azure deployment, replace the in-memory limiter with a distributed limiter such as Redis so all container replicas share the same quota state.

## API Routes

Health:

```http
GET /health/live
GET /health/ready
```

Authentication:

```http
POST /v1/auth/register
POST /v1/auth/login
```

Products:

```http
GET    /v1/products?q=apple&limit=50&offset=0
GET    /v1/products/{id}
POST   /v1/products
PUT    /v1/products/{id}
DELETE /v1/products/{id}
```

Cart:

```http
GET    /v1/cart
PUT    /v1/cart/items/{productID}
DELETE /v1/cart/items/{productID}
DELETE /v1/cart
```

Protected routes require:

```http
Authorization: Bearer <access_token>
```

## Configuration

Copy `.env.example` and set values for your environment:

```powershell
Copy-Item .env.example .env
```

Important variables:

```text
APP_ENV=development
PORT=8080
DATABASE_DSN=sqlserver://user:password@server.database.windows.net:1433?database=GinGrocery&encrypt=true
JWT_SECRET=replace-with-a-long-random-secret
JWT_ACCESS_TTL=15m
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=20
```

When `DATABASE_DSN` is empty, the app uses the in-memory repositories. That is useful for quick local development and unit tests. In `APP_ENV=production`, `DATABASE_DSN` and a real `JWT_SECRET` are required.

## Run Locally Without SQL Server

```powershell
go mod download
go run ./cmd/api
```

The API listens on `http://localhost:8080`.

## Run With SQL Server

Start a local SQL Server container:

```powershell
$env:SQLSERVER_SA_PASSWORD="Your_strong_password123"
docker compose up -d sqlserver
```

Create the database and apply `migrations/001_init.sql` using Azure Data Studio, SQL Server Management Studio, or `sqlcmd`.

Example connection string:

```powershell
$env:DATABASE_DSN="sqlserver://sa:Your_strong_password123@localhost:1433?database=GinGrocery&encrypt=disable"
$env:JWT_SECRET="local-development-secret-change-me"
go run ./cmd/api
```

For Azure SQL, use `encrypt=true` and a least-privilege database user.

## Example Requests

Register:

```powershell
Invoke-RestMethod -Method Post http://localhost:8080/v1/auth/register `
  -ContentType "application/json" `
  -Body '{"email":"buyer@example.com","password":"very-secret-password"}'
```

Create a product:

```powershell
$token = "<access_token>"
Invoke-RestMethod -Method Post http://localhost:8080/v1/products `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body '{"name":"Honeycrisp Apples","sku":"APL-HONEY-001","description":"Fresh seasonal apples","price_cents":349,"stock":24}'
```

Add the product to the cart:

```powershell
Invoke-RestMethod -Method Put http://localhost:8080/v1/cart/items/<product_id> `
  -Headers @{ Authorization = "Bearer $token" } `
  -ContentType "application/json" `
  -Body '{"quantity":2}'
```

## Docker

Build the API image:

```powershell
docker build -t gin-grocery-api .
```

Run it with environment variables:

```powershell
docker run --rm -p 8080:8080 `
  -e APP_ENV=production `
  -e JWT_SECRET="replace-with-a-long-random-secret" `
  -e DATABASE_DSN="sqlserver://user:password@server.database.windows.net:1433?database=GinGrocery&encrypt=true" `
  gin-grocery-api
```

The Dockerfile uses a multi-stage build and runs the final binary as a non-root user.

## Terraform

Azure infrastructure examples live in `terraform/`.

That configuration creates:

- Azure Container Apps for the Gin API
- Azure Container Registry for the API image
- Azure SQL Database
- Azure Service Bus for async events
- Azure Functions Flex Consumption for scheduled and event-driven background work
- Log Analytics and Application Insights

See `terraform/README.md` for the deployment flow. This is especially useful when Docker is blocked locally because Azure Container Registry can build the image in Azure with `az acr build`.

## Azure Deployment Notes

A common Azure setup is:

- Azure Container Apps or Azure App Service for Containers for the API image.
- Azure SQL Database for persistence.
- Azure Key Vault or managed app secrets for `JWT_SECRET` and `DATABASE_DSN`.
- Azure Container Registry for the Docker image.
- Application Insights or another log sink for structured JSON logs.

At deployment time:

1. Create the Azure SQL database.
2. Apply `migrations/001_init.sql`.
3. Build and push the image to Azure Container Registry.
4. Configure environment variables on the container app.
5. Expose port `8080`.
6. Point health checks at `/health/ready`.

## Tests

Run all tests:

```powershell
go test ./...
```

The tests use the in-memory repositories, so they do not need SQL Server. Current coverage includes registration/login, invalid credentials, duplicate users, protected route enforcement, product creation, cart updates, and rate limiting.

## Design Choices

Gin handlers use `ShouldBindJSON` so validation errors are returned in a consistent JSON shape instead of Gin writing a default text response.

JWTs are short-lived access tokens signed with HMAC SHA-256. Passwords are stored as bcrypt hashes.

The rate limiter uses `golang.org/x/time/rate` with a token-bucket algorithm. The limiter is deliberately middleware, not service code, because request throttling is an API policy rather than grocery business logic.

The repository interfaces keep SQL out of handlers and services. The SQL Server adapter can be replaced or expanded without changing the API contract.

Prices are stored as integer cents to avoid floating-point money errors.

## References

- Gin binding and validation: https://gin-gonic.com/en/docs/binding/
- Gin middleware: https://gin-gonic.com/en/docs/middleware/
- Microsoft Go SQL Server / Azure SQL driver guidance: https://learn.microsoft.com/en-us/azure/azure-sql/database/connect-query-go
