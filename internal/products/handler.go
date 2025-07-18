package products

import (
	"encoding/json"
	"net/http"
	"strconv"

	"inventory-system/internal"
	"inventory-system/internal/notifications"
	"inventory-system/internal/users"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validate = validator.New()

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func RegisterRoutes(r chi.Router, db *pgxpool.Pool) {
	repo := NewRepository(db)
	waToken := os.Getenv("WHATSAPP_TOKEN")
	waPhoneID := os.Getenv("WHATSAPP_PHONE_ID")
	notifier := notifications.NewNotificationService(
		&notifications.LogSender{},
		&notifications.WhatsAppSender{APIToken: waToken, PhoneID: waPhoneID},
	)
	service := NewService(repo, notifier)

	r.Route("/products", func(r chi.Router) {
		r.Use(internal.AuthMiddleware)
		r.Post("/", createProductHandler(service))
		r.Get("/", getAllProductsHandler(service))
		r.Get("/{barcode}", getProductByBarcodeHandler(service))
		r.Put("/{id}", updateProductHandler(service))
		r.With(users.RequireRole("admin", []byte("changeme"))).Delete("/{id}", deleteProductHandler(service))
		r.Post("/{barcode}/entry", stockEntryHandler(service))
		r.Post("/{barcode}/exit", stockExitHandler(service))
	})
}

// @Security ApiKeyAuth
// @Summary Create a new product
// @Tags products
// @Accept json
// @Produce json
// @Param product body Product true "Product data" example({"name":"Apple","barcode":"123456","quantity":10,"min_stock":2})
// @Success 201 {object} map[string]string "Created"
// @Failure 400 {object} map[string]string "Invalid data or duplicate barcode"
// @Router /products [post]
func createProductHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		if err := validate.Struct(&p); err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
			return
		}
		if err := s.CreateProduct(r.Context(), &p); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusCreated, nil)
	}
}

// @Security ApiKeyAuth
// @Summary List all products
// @Tags products
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param name query string false "Filter by name (partial match)"
// @Param barcode query string false "Filter by barcode (exact match)"
// @Param min_stock query int false "Filter by minimum stock"
// @Param sort query string false "Sort field (id, name, quantity, min_stock)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {array} Product "List of products" example([{...}])
// @Header 200 {int} X-Total-Count "Total number of products"
// @Router /products [get]
func getAllProductsHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Paginação
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}
		// Filtros
		name := r.URL.Query().Get("name")
		barcode := r.URL.Query().Get("barcode")
		minStock, _ := strconv.Atoi(r.URL.Query().Get("min_stock"))
		// Ordenação
		sort := r.URL.Query().Get("sort")
		order := r.URL.Query().Get("order")
		if order != "desc" {
			order = "asc"
		}
		products, total, err := s.GetProducts(r.Context(), ProductsQuery{
			Page:     page,
			Limit:    limit,
			Name:     name,
			Barcode:  barcode,
			MinStock: minStock,
			Sort:     sort,
			Order:    order,
		})
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		respondJSON(w, http.StatusOK, products)
	}
}

// @Security ApiKeyAuth
// @Summary Get product by barcode
// @Tags products
// @Produce json
// @Param barcode path string true "Barcode"
// @Success 200 {object} Product "Product data"
// @Failure 404 {object} map[string]string "Product not found"
// @Router /products/{barcode} [get]
func getProductByBarcodeHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := chi.URLParam(r, "barcode")
		product, err := s.GetProductByBarcode(r.Context(), barcode)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if product == nil {
			respondError(w, http.StatusNotFound, "Product not found")
			return
		}
		respondJSON(w, http.StatusOK, product)
	}
}

// @Security ApiKeyAuth
// @Summary Update a product
// @Tags products
// @Accept json
// @Param id path int true "Product ID"
// @Param product body Product true "Product data" example({"name":"Apple","barcode":"123456","quantity":10,"min_stock":2})
// @Success 200 {object} map[string]string "Updated"
// @Failure 400 {object} map[string]string "Invalid data"
// @Router /products/{id} [put]
func updateProductHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid ID")
			return
		}
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		if err := validate.Struct(&p); err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
			return
		}
		if err := s.UpdateProduct(r.Context(), id, &p); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, nil)
	}
}

// @Security ApiKeyAuth
// @Summary Delete a product
// @Tags products
// @Param id path int true "Product ID"
// @Success 204 {object} map[string]string "Deleted"
// @Router /products/{id} [delete]
func deleteProductHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid ID")
			return
		}
		if err := s.DeleteProduct(r.Context(), id); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusNoContent, nil)
	}
}

// @Security ApiKeyAuth
// @Summary Stock entry
// @Tags stock
// @Accept json
// @Param barcode path string true "Barcode"
// @Param body body StockRequest true "Quantity" example({"quantity":5})
// @Success 200 {object} map[string]string "Stock updated"
// @Router /products/{barcode}/entry [post]
func stockEntryHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := chi.URLParam(r, "barcode")
		var req StockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		if err := validate.Struct(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
			return
		}
		if err := s.StockEntry(r.Context(), barcode, req.Quantity); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, nil)
	}
}

// @Security ApiKeyAuth
// @Summary Stock exit
// @Tags stock
// @Accept json
// @Param barcode path string true "Barcode"
// @Param body body StockRequest true "Quantity" example({"quantity":5})
// @Success 200 {object} map[string]string "Stock updated"
// @Failure 400 {object} map[string]string "Insufficient stock"
// @Router /products/{barcode}/exit [post]
func stockExitHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcode := chi.URLParam(r, "barcode")
		var req StockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		if err := validate.Struct(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
			return
		}
		if err := s.StockExit(r.Context(), barcode, req.Quantity); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, nil)
	}
}
