package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ReadinessCheck func(context.Context) error

type HealthHandler struct {
	checks []ReadinessCheck
}

func NewHealthHandler(checks ...ReadinessCheck) *HealthHandler {
	return &HealthHandler{checks: checks}
}

func (h *HealthHandler) Live(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	for _, check := range h.checks {
		if err := check(ctx); err != nil {
			_ = c.Error(err)
			respond(c, http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
	}

	respond(c, http.StatusOK, gin.H{"status": "ready"})
}
