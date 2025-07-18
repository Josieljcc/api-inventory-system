package users

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jwtSecret = []byte(getEnv("JWT_SECRET", "changeme"))

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

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
	svc := NewService(repo)
	r.With(RateLimitMiddleware(5, time.Minute)).Post("/register", registerHandler(svc))
	r.With(RateLimitMiddleware(5, time.Minute)).Post("/login", loginHandler(svc))
	r.With(RateLimitMiddleware(5, time.Minute)).Post("/refresh", refreshHandler(svc))
}

// @Summary Register a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body LoginRequest true "User credentials" example({"username":"johndoe","password":"secret"})
// @Success 201 {object} map[string]string "Created"
// @Failure 400 {object} map[string]string "Invalid data or duplicate username"
// @Router /register [post]
func registerHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		if req.Username == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "Username and password are required")
			return
		}
		// Check if this is the first user (admin)
		role := "user"
		if isFirstUser(s) {
			role = "admin"
		}
		if err := s.Register(r.Context(), req.Username, req.Password, role); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondJSON(w, http.StatusCreated, nil)
	}
}

func isFirstUser(s *Service) bool {
	// Try to get any user; if not found, it's the first
	_, err := s.GetByUsername(context.Background(), "admin-check")
	return err != nil // If error, assume no users
}

// @Summary Authenticate a user
// @Tags users
// @Accept json
// @Produce json
// @Param user body LoginRequest true "User credentials" example({"username":"johndoe","password":"secret"})
// @Success 200 {object} map[string]string "JWT and refresh token" example({"token":"<jwt>","refresh_token":"<refresh>"})
// @Failure 400 {object} map[string]string "Invalid data"
// @Failure 401 {object} map[string]string "Invalid username or password"
// @Router /login [post]
func loginHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid data")
			return
		}
		u, err := s.Authenticate(r.Context(), req.Username, req.Password)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":      u.ID,
			"username": u.Username,
			"role":     u.Role,
			"exp":      time.Now().Add(15 * time.Minute).Unix(),
		})
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}
		refreshToken, err := s.GenerateRefreshToken(r.Context(), u.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{
			"token":         tokenString,
			"refresh_token": refreshToken,
		})
	}
}

// @Summary Refresh access token
// @Tags users
// @Accept json
// @Produce json
// @Param refresh_token body string true "Refresh token" example({"refresh_token":"<refresh>"})
// @Success 200 {object} map[string]string "New JWT and refresh token" example({"token":"<jwt>","refresh_token":"<refresh>"})
// @Failure 400 {object} map[string]string "Invalid refresh token"
// @Failure 401 {object} map[string]string "Invalid or expired refresh token"
// @Router /refresh [post]
func refreshHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			respondError(w, http.StatusBadRequest, "Invalid refresh token")
			return
		}
		// Validar e rotacionar refresh token
		newRefreshToken, err := s.RotateRefreshToken(r.Context(), req.RefreshToken)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
			return
		}
		rt, _ := s.ValidateRefreshToken(r.Context(), newRefreshToken)
		if rt == nil {
			respondError(w, http.StatusUnauthorized, "User not found")
			return
		}
		// Gerar novo access token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": rt.UserID,
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		})
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{
			"token":         tokenString,
			"refresh_token": newRefreshToken,
		})
	}
}
