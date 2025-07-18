package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@db:5432/inventory?sslmode=disable"
	}
	var err error
	testDB, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic(err)
	}
	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

func TestLoginErrorCases(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	// Criar usuário para teste
	_ = svc.Register(context.Background(), "testuser", "testpass", "user")

	// Teste login sem username
	body := bytes.NewBufferString(`{"password":"testpass"}`)
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("login sem username: esperado 400, veio %d", resp.Code)
	}

	// Teste login sem password
	body = bytes.NewBufferString(`{"username":"testuser"}`)
	req = httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("login sem password: esperado 400, veio %d", resp.Code)
	}

	// Teste login com JSON malformado
	body = bytes.NewBufferString(`{"username":"testuser","password":"testpass"`)
	req = httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("JSON malformado: esperado 400, veio %d", resp.Code)
	}

	// Teste login com usuário inexistente
	body = bytes.NewBufferString(`{"username":"inexistent","password":"testpass"}`)
	req = httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("usuário inexistente: esperado 401, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Invalid username or password")) {
		t.Error("mensagem de erro incorreta para usuário inexistente")
	}

	// Teste login com senha incorreta
	body = bytes.NewBufferString(`{"username":"testuser","password":"wrongpass"}`)
	req = httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("senha incorreta: esperado 401, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Invalid username or password")) {
		t.Error("mensagem de erro incorreta para senha incorreta")
	}
}

func TestDatabaseErrorCases(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)

	// Teste buscar usuário inexistente
	user, err := svc.GetByUsername(context.Background(), "inexistent")
	if err == nil {
		t.Error("buscar usuário inexistente deveria retornar erro")
	}
	if user != nil {
		t.Error("usuário inexistente deveria retornar nil")
	}

	// Teste autenticar usuário inexistente
	user, err = svc.Authenticate(context.Background(), "inexistent", "testpass")
	if err == nil {
		t.Error("autenticar usuário inexistente deveria retornar erro")
	}
	if user != nil {
		t.Error("usuário inexistente deveria retornar nil")
	}
}

func TestRegisterAndLogin(t *testing.T) {
	cleanTable(t)
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	// Registro
	body := bytes.NewBufferString(`{"username":"testuser","password":"testpass"}`)
	req := httptest.NewRequest("POST", "/register", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d", resp.Code)
	}

	// Login
	body = bytes.NewBufferString(`{"username":"testuser","password":"testpass"}`)
	req = httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("esperado 200, veio %d", resp.Code)
	}
	var lr LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}
	if !strings.HasPrefix(lr.Token, "eyJ") {
		t.Error("token JWT não retornado corretamente")
	}
}

func cleanTable(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "DELETE FROM users")
	if err != nil {
		t.Fatalf("erro ao limpar tabela: %v", err)
	}
}
