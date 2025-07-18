// @title Inventory System API
// @version 1.0
// @description API para gerenciamento de estoque e produtos.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
package main

import (
	"log"
	"net/http"
	"os"

	_ "inventory-system/docs"
	"inventory-system/internal"
	"inventory-system/internal/database"
	"inventory-system/internal/products"
	"inventory-system/internal/users"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @Summary Health check
// @Tags sistema
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env")
	}
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL não definida")
	}
	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("Erro ao conectar no banco: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Erro ao rodar migrations: %v", err)
	}

	r := chi.NewRouter()
	r.Use(internal.CORSMiddleware)

	r.Get("/health", healthHandler)

	r.Get("/swagger/*", httpSwagger.WrapHandler)
	users.RegisterRoutes(r, db)
	products.RegisterRoutes(r, db)

	log.Println("Servidor rodando na porta 8080...")
	http.ListenAndServe(":8080", r)
}
