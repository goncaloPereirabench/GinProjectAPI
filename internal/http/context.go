package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	contextUserID = "user_id"
	contextEmail  = "email"
	contextRole   = "role"
)

func currentUserID(c *gin.Context) (uuid.UUID, error) {
	value, ok := c.Get(contextUserID)
	if !ok {
		return uuid.Nil, errors.New("user id missing from context")
	}

	id, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user id has unexpected type")
	}
	return id, nil
}
