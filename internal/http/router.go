package http

import (
	"log/slog"
	"net/http"

	"ginprojectapi/internal/config"
	"ginprojectapi/internal/service"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, services service.Services, logger *slog.Logger, readinessChecks ...ReadinessCheck) http.Handler {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		RequestID(),
		StructuredLogger(logger),
		gin.Recovery(),
		SecurityHeaders(),
		CORS(cfg.CORS),
	)

	health := NewHealthHandler(readinessChecks...)
	router.GET("/health/live", health.Live)
	router.GET("/health/ready", health.Ready)

	v1 := router.Group("/v1")
	v1.Use(RateLimit(cfg.RateLimit, services.JWT))

	authHandler := NewAuthHandler(services.Auth)
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)

	productHandler := NewProductHandler(services.Product)
	v1.GET("/products", productHandler.List)
	v1.GET("/products/:id", productHandler.Get)

	protected := v1.Group("")
	protected.Use(AuthRequired(services.JWT))
	protected.POST("/products", productHandler.Create)
	protected.PUT("/products/:id", productHandler.Update)
	protected.DELETE("/products/:id", productHandler.Delete)

	cartHandler := NewCartHandler(services.Cart)
	protected.GET("/cart", cartHandler.Get)
	protected.PUT("/cart/items/:productID", cartHandler.SetItem)
	protected.DELETE("/cart/items/:productID", cartHandler.RemoveItem)
	protected.DELETE("/cart", cartHandler.Clear)

	return router
}
