package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ginprojectapi/internal/config"
	"ginprojectapi/internal/service"
	"ginprojectapi/internal/store/memory"
)

func TestAuthProductAndCartFlow(t *testing.T) {
	router := newTestRouter(60, 20)

	register := doJSON(t, router, nethttp.MethodPost, "/v1/auth/register", map[string]any{
		"email":    "buyer@example.com",
		"password": "very-secret-password",
	}, "")
	if register.Code != nethttp.StatusCreated {
		t.Fatalf("expected register status 201, got %d: %s", register.Code, register.Body.String())
	}

	var auth authResponse
	decodeJSON(t, register.Body.Bytes(), &auth)
	if auth.Token.AccessToken == "" {
		t.Fatal("expected access token")
	}

	create := doJSON(t, router, nethttp.MethodPost, "/v1/products", map[string]any{
		"name":        "Honeycrisp Apples",
		"sku":         "APL-HONEY-001",
		"description": "Fresh seasonal apples",
		"price_cents": 349,
		"stock":       24,
	}, auth.Token.AccessToken)
	if create.Code != nethttp.StatusCreated {
		t.Fatalf("expected product status 201, got %d: %s", create.Code, create.Body.String())
	}

	var product productResponse
	decodeJSON(t, create.Body.Bytes(), &product)
	if product.ID == "" || product.SKU != "APL-HONEY-001" {
		t.Fatalf("unexpected product response: %+v", product)
	}

	cart := doJSON(t, router, nethttp.MethodPut, "/v1/cart/items/"+product.ID, map[string]any{
		"quantity": 2,
	}, auth.Token.AccessToken)
	if cart.Code != nethttp.StatusOK {
		t.Fatalf("expected cart status 200, got %d: %s", cart.Code, cart.Body.String())
	}

	var cartBody cartResponse
	decodeJSON(t, cart.Body.Bytes(), &cartBody)
	if cartBody.TotalCents != 698 || len(cartBody.Items) != 1 {
		t.Fatalf("unexpected cart response: %+v", cartBody)
	}
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	router := newTestRouter(60, 20)
	response := doJSON(t, router, nethttp.MethodPost, "/v1/products", map[string]any{
		"name":        "Milk",
		"sku":         "DAIRY-MILK",
		"price_cents": 199,
		"stock":       10,
	}, "")

	if response.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestRegisterRejectsMalformedJSON(t *testing.T) {
	router := newTestRouter(60, 20)
	request := httptest.NewRequest(nethttp.MethodPost, "/v1/auth/register", bytes.NewBufferString("{email:broken}"))
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}

	var body errorResponse
	decodeJSON(t, response.Body.Bytes(), &body)
	if body.Error != "invalid_json" {
		t.Fatalf("expected invalid_json, got %+v", body)
	}
}

func TestCORSPreflightAllowsConfiguredFrontendOrigin(t *testing.T) {
	router := newTestRouter(60, 20)
	request := httptest.NewRequest(nethttp.MethodOptions, "/v1/cart", nil)
	request.Header.Set("Origin", "https://frontend.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example.com" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(strings.ToLower(got), "authorization") {
		t.Fatalf("expected authorization to be allowed, got %q", got)
	}
}

func TestCORSRejectsUnknownBrowserOrigin(t *testing.T) {
	router := newTestRouter(60, 20)
	request := httptest.NewRequest(nethttp.MethodOptions, "/v1/cart", nil)
	request.Header.Set("Origin", "https://unknown.example.com")
	request.Header.Set("Access-Control-Request-Method", "GET")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}
}

func TestRateLimitReturnsTooManyRequests(t *testing.T) {
	router := newTestRouter(1, 1)

	first := doJSON(t, router, nethttp.MethodGet, "/v1/products", nil, "")
	if first.Code != nethttp.StatusOK {
		t.Fatalf("expected first request 200, got %d", first.Code)
	}

	second := doJSON(t, router, nethttp.MethodGet, "/v1/products", nil, "")
	if second.Code != nethttp.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", second.Code)
	}
}

func TestReadyReturnsUnavailableWhenDependencyFails(t *testing.T) {
	router := newTestRouterWithReadiness(func(context.Context) error {
		return errors.New("database unavailable")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(nethttp.MethodGet, "/health/ready", nil)
	router.ServeHTTP(response, request)

	if response.Code != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", response.Code, response.Body.String())
	}
}

func newTestRouter(requestsPerMinute, burst int) nethttp.Handler {
	return newTestRouterWithReadiness(nil, requestsPerMinute, burst)
}

func newTestRouterWithReadiness(check ReadinessCheck, limits ...int) nethttp.Handler {
	requestsPerMinute := 60
	burst := 20
	if len(limits) > 0 {
		requestsPerMinute = limits[0]
	}
	if len(limits) > 1 {
		burst = limits[1]
	}

	repositories := memory.NewRepositories()
	jwtManager := service.NewJWTManager("test-secret", "test-suite", 15*time.Minute)
	services := service.Services{
		Auth:    service.NewAuthService(repositories.Users, jwtManager),
		Product: service.NewProductService(repositories.Products),
		Cart:    service.NewCartService(repositories.Products, repositories.Carts),
		JWT:     jwtManager,
	}
	cfg := config.Config{
		Environment: "test",
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: requestsPerMinute,
			Burst:             burst,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"https://frontend.example.com"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders: []string{"X-Request-ID"},
			MaxAge:         12 * time.Hour,
		},
	}

	if check == nil {
		return NewRouter(cfg, services, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
	return NewRouter(cfg, services, slog.New(slog.NewTextHandler(io.Discard, nil)), check)
}

func doJSON(t *testing.T, handler nethttp.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeJSON(t *testing.T, raw []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(raw))
	}
}
