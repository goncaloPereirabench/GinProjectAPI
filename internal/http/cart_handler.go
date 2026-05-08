package http

import (
	"net/http"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/service"
	"ginprojectapi/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler struct {
	carts *service.CartService
}

type cartItemRequest struct {
	Quantity int `json:"quantity" binding:"gte=0,lte=999"`
}

type cartResponse struct {
	UserID     string             `json:"user_id"`
	Items      []cartItemResponse `json:"items"`
	TotalCents int64              `json:"total_cents"`
}

type cartItemResponse struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
}

func NewCartHandler(carts *service.CartService) *CartHandler {
	return &CartHandler{carts: carts}
}

func (h *CartHandler) Get(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		respondError(c, err)
		return
	}

	cart, err := h.carts.Get(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, toCartResponse(cart))
}

func (h *CartHandler) SetItem(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		respondError(c, err)
		return
	}

	productID, err := uuid.Parse(c.Param("productID"))
	if err != nil {
		respondError(c, store.ErrInvalid)
		return
	}

	var request cartItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, err)
		return
	}

	cart, err := h.carts.SetItem(c.Request.Context(), userID, productID, request.Quantity)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, toCartResponse(cart))
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		respondError(c, err)
		return
	}

	productID, err := uuid.Parse(c.Param("productID"))
	if err != nil {
		respondError(c, store.ErrInvalid)
		return
	}

	cart, err := h.carts.RemoveItem(c.Request.Context(), userID, productID)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, toCartResponse(cart))
}

func (h *CartHandler) Clear(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.carts.Clear(c.Request.Context(), userID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toCartResponse(cart domain.Cart) cartResponse {
	response := cartResponse{
		UserID:     cart.UserID.String(),
		Items:      make([]cartItemResponse, 0, len(cart.Items)),
		TotalCents: cart.TotalCents,
	}
	for _, item := range cart.Items {
		response.Items = append(response.Items, cartItemResponse{
			ProductID:   item.ProductID.String(),
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			LineTotal:   item.LineTotal,
		})
	}
	return response
}
