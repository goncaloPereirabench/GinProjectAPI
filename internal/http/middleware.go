package http

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowedOrigins[normalized] = struct{}{}
		}
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := int(cfg.MaxAge.Seconds())

	return func(c *gin.Context) {
		origin := normalizeOrigin(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}

		c.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
		if _, ok := allowedOrigins[origin]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, errorResponse{
				Error:   "origin_not_allowed",
				Message: "this origin is not allowed to call the API",
			})
			return
		}

		// Browsers require these headers before they allow frontend JavaScript
		// to send Authorization headers or read selected response headers.
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Expose-Headers", exposedHeaders)
		c.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

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

func normalizeOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}

	host := strings.ToLower(parsed.Host)
	if hostname, port, err := net.SplitHostPort(host); err == nil {
		host = net.JoinHostPort(strings.ToLower(hostname), port)
	}
	return parsed.Scheme + "://" + host
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
