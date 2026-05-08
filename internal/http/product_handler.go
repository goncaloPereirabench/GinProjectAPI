package http

import (
	"net/http"
	"strconv"

	"ginprojectapi/internal/domain"
	"ginprojectapi/internal/service"
	"ginprojectapi/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	products *service.ProductService
}

type productRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=120"`
	SKU         string `json:"sku" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=500"`
	PriceCents  int64  `json:"price_cents" binding:"gte=0"`
	Stock       int    `json:"stock" binding:"gte=0"`
	Active      *bool  `json:"active"`
}

type productResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Stock       int    `json:"stock"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func NewProductHandler(products *service.ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var request productRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, err)
		return
	}

	product, err := h.products.Create(c.Request.Context(), productInput(request))
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusCreated, toProductResponse(product))
}

func (h *ProductHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	products, err := h.products.List(c.Request.Context(), store.ProductFilter{
		Query:           c.Query("q"),
		IncludeInactive: c.Query("include_inactive") == "true",
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response := make([]productResponse, 0, len(products))
	for _, product := range products {
		response = append(response, toProductResponse(product))
	}
	respond(c, http.StatusOK, gin.H{"items": response, "count": len(response)})
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, store.ErrInvalid)
		return
	}

	product, err := h.products.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, store.ErrInvalid)
		return
	}

	var request productRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, err)
		return
	}

	product, err := h.products.Update(c.Request.Context(), id, productInput(request))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, store.ErrInvalid)
		return
	}

	if err := h.products.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func productInput(request productRequest) service.ProductInput {
	active := true
	if request.Active != nil {
		active = *request.Active
	}
	return service.ProductInput{
		Name:        request.Name,
		SKU:         request.SKU,
		Description: request.Description,
		PriceCents:  request.PriceCents,
		Stock:       request.Stock,
		Active:      active,
	}
}

func toProductResponse(product domain.Product) productResponse {
	return productResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		SKU:         product.SKU,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Stock:       product.Stock,
		Active:      product.Active,
		CreatedAt:   product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   product.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
