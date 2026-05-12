package http

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"ginprojectapi/internal/config"
	"ginprojectapi/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func StructuredLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.Writer.Header().Get("X-Request-ID"),
			"client_ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		logger.Info("http request",
			attrs...,
		)
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func AuthRequired(jwtManager *service.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{
				Error:   "unauthorized",
				Message: "a bearer token is required",
			})
			return
		}

		claims, err := jwtManager.Parse(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{
				Error:   "unauthorized",
				Message: "the bearer token is invalid or expired",
			})
			return
		}

		c.Set(contextUserID, claims.UserID)
		c.Set(contextEmail, claims.Email)
		c.Set(contextRole, claims.Role)
		c.Next()
	}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(cfg config.RateLimitConfig, jwtManager *service.JWTManager) gin.HandlerFunc {
	requestsPerMinute := cfg.RequestsPerMinute
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = requestsPerMinute
	}

	var mu sync.Mutex
	visitors := make(map[string]*visitor)
	limit := rate.Every(time.Minute / time.Duration(requestsPerMinute))

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for key, visitor := range visitors {
				if time.Since(visitor.lastSeen) > 3*time.Minute {
					delete(visitors, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := rateLimitKey(c, jwtManager)

		mu.Lock()
		v, exists := visitors[key]
		if !exists {
			v = &visitor{limiter: rate.NewLimiter(limit, burst)}
			visitors[key] = v
		}
		v.lastSeen = time.Now()
		allowed := v.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errorResponse{
				Error:   "rate_limited",
				Message: "too many requests; please retry later",
			})
			return
		}

		c.Next()
	}
}

func rateLimitKey(c *gin.Context, jwtManager *service.JWTManager) string {
	if token := bearerToken(c.GetHeader("Authorization")); token != "" {
		if claims, err := jwtManager.Parse(token); err == nil {
			return "user:" + claims.UserID.String()
		}
	}
	return "ip:" + c.ClientIP()
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
