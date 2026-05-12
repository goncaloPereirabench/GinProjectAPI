package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"ginprojectapi/internal/service"
	"ginprojectapi/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func respond(c *gin.Context, status int, value any) {
	c.JSON(status, value)
}

func respondError(c *gin.Context, err error) {
	_ = c.Error(err)

	status := http.StatusInternalServerError
	code := "internal_error"
	message := "something went wrong"

	switch {
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "the requested resource was not found"
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = "the resource already exists"
	case errors.Is(err, store.ErrInvalid):
		status = http.StatusBadRequest
		code = "invalid_request"
		message = "the request is invalid for the current resource state"
	case errors.Is(err, service.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		code = "invalid_credentials"
		message = "email or password is incorrect"
	default:
		var validationErrors validator.ValidationErrors
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &validationErrors) {
			status = http.StatusBadRequest
			code = "validation_failed"
			message = "one or more request fields are invalid"
		} else if errors.As(err, &syntaxError) || errors.As(err, &typeError) || errors.Is(err, io.EOF) {
			status = http.StatusBadRequest
			code = "invalid_json"
			message = "request body must be valid JSON"
		}
	}

	c.JSON(status, errorResponse{Error: code, Message: message})
}
