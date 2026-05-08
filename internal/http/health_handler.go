package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Live(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"status": "ready"})
}
