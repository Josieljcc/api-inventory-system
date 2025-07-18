package products

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"bytes"
	"net/http"
	"net/http/httptest"

	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool

var jwtSecret = []byte("changeme")

func generateValidToken() string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      1,
		"username": "testuser",
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

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

func cleanTable(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("erro ao limpar tabela: %v", err)
	}
}

func TestCreateAndGetProduct(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	p := &Product{Name: "Produto Teste", Barcode: "123456", Quantity: 10, MinStock: 2}
	err := svc.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatalf("erro ao criar produto: %v", err)
	}
	if p.ID == 0 {
		t.Error("ID não foi preenchido")
	}
	prod, err := svc.GetProductByBarcode(context.Background(), "123456")
	if err != nil || prod == nil {
		t.Fatalf("erro ao buscar produto: %v", err)
	}
	if prod.Name != "Produto Teste" {
		t.Errorf("Nome incorreto: %v", prod.Name)
	}
}

func TestGetAllProducts(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	_ = svc.CreateProduct(context.Background(), &Product{Name: "P1", Barcode: "b1", Quantity: 1, MinStock: 1})
	_ = svc.CreateProduct(context.Background(), &Product{Name: "P2", Barcode: "b2", Quantity: 2, MinStock: 1})
	prods, _, err := svc.GetProducts(context.Background(), ProductsQuery{})
	if err != nil {
		t.Fatalf("erro ao buscar todos: %v", err)
	}
	if len(prods) != 2 {
		t.Errorf("esperado 2 produtos, veio %d", len(prods))
	}
}

func TestUpdateProduct(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	p := &Product{Name: "P", Barcode: "b", Quantity: 1, MinStock: 1}
	_ = svc.CreateProduct(context.Background(), p)
	p.Name = "Novo Nome"
	p.Quantity = 99
	err := svc.UpdateProduct(context.Background(), p.ID, p)
	if err != nil {
		t.Fatalf("erro ao atualizar: %v", err)
	}
	prod, _ := svc.GetProductByBarcode(context.Background(), "b")
	if prod.Name != "Novo Nome" || prod.Quantity != 99 {
		t.Errorf("update não refletiu: %+v", prod)
	}
}

func TestDeleteProduct(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	p := &Product{Name: "P", Barcode: "b", Quantity: 1, MinStock: 1}
	_ = svc.CreateProduct(context.Background(), p)
	err := svc.DeleteProduct(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("erro ao deletar: %v", err)
	}
	prod, _ := svc.GetProductByBarcode(context.Background(), "b")
	if prod != nil {
		t.Error("produto não foi deletado")
	}
}

func TestStockEntryAndExit(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	p := &Product{Name: "P", Barcode: "b", Quantity: 10, MinStock: 1}
	_ = svc.CreateProduct(context.Background(), p)
	err := svc.StockEntry(context.Background(), "b", 5)
	if err != nil {
		t.Fatalf("erro ao dar entrada: %v", err)
	}
	prod, _ := svc.GetProductByBarcode(context.Background(), "b")
	if prod.Quantity != 15 {
		t.Errorf("entrada não refletiu: %d", prod.Quantity)
	}
	err = svc.StockExit(context.Background(), "b", 10)
	if err != nil {
		t.Fatalf("erro ao dar saída: %v", err)
	}
	prod, _ = svc.GetProductByBarcode(context.Background(), "b")
	if prod.Quantity != 5 {
		t.Errorf("saída não refletiu: %d", prod.Quantity)
	}
	// Testar saída maior que estoque
	err = svc.StockExit(context.Background(), "b", 99)
	if err == nil {
		t.Error("esperava erro de estoque insuficiente")
	}
}

func TestCreateProductValidation(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)
	// Produto sem nome
	body := bytes.NewBufferString(`{"barcode":"abc","quantity":1,"min_stock":1}`)
	req := httptest.NewRequest("POST", "/products", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`"error"`)) {
		t.Error("resposta não contém campo 'error'")
	}
}

func TestStockEntryValidation(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)
	// Quantidade inválida
	body := bytes.NewBufferString(`{"quantity":0}`)
	req := httptest.NewRequest("POST", "/products/abc/entry", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`"error"`)) {
		t.Error("resposta não contém campo 'error'")
	}
}

func TestAuthMiddleware_Required(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	endpoints := []struct {
		method, path, body string
	}{
		{"GET", "/products", ""},
		{"POST", "/products", `{"name":"P","barcode":"b","quantity":1,"min_stock":1}`},
		{"GET", "/products/abc", ""},
		{"PUT", "/products/1", `{"name":"P","barcode":"b","quantity":1,"min_stock":1}`},
		{"DELETE", "/products/1", ""},
		{"POST", "/products/abc/entry", `{"quantity":1}`},
		{"POST", "/products/abc/exit", `{"quantity":1}`},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sem token: esperado 401, veio %d", ep.method, ep.path, resp.Code)
		}
		// Token inválido
		req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token_invalido")
		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("%s %s token inválido: esperado 401, veio %d", ep.method, ep.path, resp.Code)
		}
		// Token válido
		req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generateValidToken())
		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code == http.StatusUnauthorized {
			t.Errorf("%s %s token válido: não deveria retornar 401", ep.method, ep.path)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("/health esperado 200, veio %d", resp.Code)
	}
	if resp.Body.String() != `{"status":"ok"}` {
		t.Errorf("/health resposta inesperada: %s", resp.Body.String())
	}
}

func TestGetProductsWithPaginationAndFilters(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)
	// Criar produtos de teste
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Apple", Barcode: "123", Quantity: 10, MinStock: 5})
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Banana", Barcode: "456", Quantity: 5, MinStock: 2})
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Orange", Barcode: "789", Quantity: 15, MinStock: 3})

	// Teste paginação
	products, total, err := svc.GetProducts(context.Background(), ProductsQuery{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("erro ao buscar produtos: %v", err)
	}
	if len(products) != 2 {
		t.Errorf("esperado 2 produtos, veio %d", len(products))
	}
	if total != 3 {
		t.Errorf("total esperado 3, veio %d", total)
	}

	// Teste filtro por nome
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Name: "Apple"})
	if err != nil {
		t.Fatalf("erro ao buscar produtos: %v", err)
	}
	if len(products) != 1 {
		t.Errorf("esperado 1 produto, veio %d", len(products))
	}
	if products[0].Name != "Apple" {
		t.Errorf("nome esperado Apple, veio %s", products[0].Name)
	}

	// Teste filtro por barcode
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Barcode: "456"})
	if err != nil {
		t.Fatalf("erro ao buscar produtos: %v", err)
	}
	if len(products) != 1 {
		t.Errorf("esperado 1 produto, veio %d", len(products))
	}
	if products[0].Barcode != "456" {
		t.Errorf("barcode esperado 456, veio %s", products[0].Barcode)
	}

	// Teste filtro por min_stock
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{MinStock: 5})
	if err != nil {
		t.Fatalf("erro ao buscar produtos: %v", err)
	}
	if len(products) != 1 {
		t.Errorf("esperado 1 produto, veio %d", len(products))
	}
	if products[0].MinStock < 5 {
		t.Errorf("min_stock esperado >= 5, veio %d", products[0].MinStock)
	}

	// Teste ordenação
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Sort: "name", Order: "asc"})
	if err != nil {
		t.Fatalf("erro ao buscar produtos: %v", err)
	}
	if len(products) < 2 {
		t.Errorf("esperado pelo menos 2 produtos, veio %d", len(products))
	}
	if products[0].Name > products[1].Name {
		t.Errorf("ordenação incorreta: %s > %s", products[0].Name, products[1].Name)
	}
}

func TestGetProductsHTTP(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	// Teste sem parâmetros
	req := httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Errorf("esperado 200, veio %d", resp.Code)
	}
	if resp.Header().Get("X-Total-Count") == "" {
		t.Error("header X-Total-Count não encontrado")
	}

	// Teste com parâmetros
	req = httptest.NewRequest("GET", "/products?page=1&limit=5&name=test&sort=name&order=asc", nil)
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Errorf("esperado 200, veio %d", resp.Code)
	}
}

func TestAuthErrorCases(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	// Teste sem header Authorization
	req := httptest.NewRequest("GET", "/products", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("sem Authorization: esperado 401, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Token ausente ou inválido")) {
		t.Error("mensagem de erro incorreta para token ausente")
	}

	// Teste com header Authorization vazio
	req = httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Authorization vazio: esperado 401, veio %d", resp.Code)
	}

	// Teste com Bearer sem token
	req = httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "Bearer ")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Bearer sem token: esperado 401, veio %d", resp.Code)
	}

	// Teste com token malformado (não JWT)
	req = httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt-token")
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("token malformado: esperado 401, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Token inválido")) {
		t.Error("mensagem de erro incorreta para token inválido")
	}

	// Teste com JWT expirado
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      1,
		"username": "testuser",
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // expirado
	})
	expiredTokenString, _ := expiredToken.SignedString(jwtSecret)
	req = httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTokenString)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("token expirado: esperado 401, veio %d", resp.Code)
	}

	// Teste com JWT com assinatura incorreta
	wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      1,
		"username": "testuser",
		"exp":      time.Now().Add(1 * time.Hour).Unix(),
	})
	wrongSecretTokenString, _ := wrongSecretToken.SignedString([]byte("wrong-secret"))
	req = httptest.NewRequest("GET", "/products", nil)
	req.Header.Set("Authorization", "Bearer "+wrongSecretTokenString)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("assinatura incorreta: esperado 401, veio %d", resp.Code)
	}
}

func TestValidationErrorCases(t *testing.T) {
	r := chi.NewRouter()
	RegisterRoutes(r, testDB)

	// Teste produto sem nome
	body := bytes.NewBufferString(`{"barcode":"123","quantity":1,"min_stock":1}`)
	req := httptest.NewRequest("POST", "/products", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("produto sem nome: esperado 400, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Validation failed")) {
		t.Error("mensagem de erro incorreta para validação")
	}

	// Teste produto sem barcode
	body = bytes.NewBufferString(`{"name":"Test Product","quantity":1,"min_stock":1}`)
	req = httptest.NewRequest("POST", "/products", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("produto sem barcode: esperado 400, veio %d", resp.Code)
	}

	// Teste JSON malformado
	body = bytes.NewBufferString(`{"name":"Test","barcode":"123","quantity":1,"min_stock":1`)
	req = httptest.NewRequest("POST", "/products", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("JSON malformado: esperado 400, veio %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Invalid data")) {
		t.Error("mensagem de erro incorreta para JSON malformado")
	}

	// Teste quantidade inválida na entrada de estoque
	body = bytes.NewBufferString(`{"quantity":-1}`)
	req = httptest.NewRequest("POST", "/products/123/entry", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("quantidade negativa: esperado 400, veio %d", resp.Code)
	}

	// Teste quantidade zero na entrada de estoque
	body = bytes.NewBufferString(`{"quantity":0}`)
	req = httptest.NewRequest("POST", "/products/123/entry", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generateValidToken())
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("quantidade zero: esperado 400, veio %d", resp.Code)
	}
}

func TestDatabaseErrorCases(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)

	// Teste busca produto inexistente
	product, err := svc.GetProductByBarcode(context.Background(), "inexistent")
	if err != nil {
		t.Errorf("erro inesperado ao buscar produto inexistente: %v", err)
	}
	if product != nil {
		t.Error("produto inexistente deveria retornar nil")
	}

	// Teste atualizar produto inexistente
	p := &Product{Name: "Test", Barcode: "123", Quantity: 1, MinStock: 1}
	err = svc.UpdateProduct(context.Background(), 999, p)
	if err == nil {
		t.Error("atualizar produto inexistente deveria retornar erro")
	}

	// Teste deletar produto inexistente
	err = svc.DeleteProduct(context.Background(), 999)
	if err == nil {
		t.Error("deletar produto inexistente deveria retornar erro")
	}

	// Teste saída de estoque de produto inexistente
	err = svc.StockExit(context.Background(), "inexistent", 1)
	if err == nil {
		t.Error("saída de estoque de produto inexistente deveria retornar erro")
	}
	if !strings.Contains(err.Error(), "Insufficient stock or product not found") {
		t.Errorf("mensagem de erro incorreta: %v", err)
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)

	// Criar produtos para teste
	for i := 1; i <= 25; i++ {
		_ = svc.CreateProduct(context.Background(), &Product{
			Name:     fmt.Sprintf("Product %d", i),
			Barcode:  fmt.Sprintf("barcode%d", i),
			Quantity: i,
			MinStock: 1,
		})
	}

	// Teste página vazia
	products, total, err := svc.GetProducts(context.Background(), ProductsQuery{Page: 10, Limit: 10})
	if err != nil {
		t.Fatalf("erro ao buscar página vazia: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("página vazia deveria retornar 0 produtos, veio %d", len(products))
	}
	if total != 25 {
		t.Errorf("total esperado 25, veio %d", total)
	}

	// Teste limite máximo
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Page: 1, Limit: 200})
	if err != nil {
		t.Fatalf("erro ao buscar com limite alto: %v", err)
	}
	if len(products) != 20 { // deve ser limitado a 20
		t.Errorf("deveria ser limitado a 20 produtos, veio %d", len(products))
	}

	// Teste página negativa
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Page: -1, Limit: 10})
	if err != nil {
		t.Fatalf("erro ao buscar página negativa: %v", err)
	}
	if len(products) != 10 {
		t.Errorf("página negativa deveria retornar 10 produtos, veio %d", len(products))
	}
}

func TestFilterEdgeCases(t *testing.T) {
	cleanTable(t)
	repo := NewRepository(testDB)
	svc := NewService(repo)

	// Criar produtos para teste
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Apple", Barcode: "123", Quantity: 10, MinStock: 5})
	_ = svc.CreateProduct(context.Background(), &Product{Name: "Banana", Barcode: "456", Quantity: 5, MinStock: 2})

	// Teste filtro por nome inexistente
	products, _, err := svc.GetProducts(context.Background(), ProductsQuery{Name: "Inexistent"})
	if err != nil {
		t.Fatalf("erro ao buscar nome inexistente: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("nome inexistente deveria retornar 0 produtos, veio %d", len(products))
	}

	// Teste filtro por barcode inexistente
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{Barcode: "999"})
	if err != nil {
		t.Fatalf("erro ao buscar barcode inexistente: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("barcode inexistente deveria retornar 0 produtos, veio %d", len(products))
	}

	// Teste filtro por min_stock alto
	products, _, err = svc.GetProducts(context.Background(), ProductsQuery{MinStock: 10})
	if err != nil {
		t.Fatalf("erro ao buscar min_stock alto: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("min_stock alto deveria retornar 0 produtos, veio %d", len(products))
	}
}

type mockProductRepo struct {
	products map[string]*Product
	fail     bool
}

func (m *mockProductRepo) CreateProduct(ctx context.Context, p *Product) error {
	if m.fail {
		return fmt.Errorf("db error")
	}
	if _, exists := m.products[p.Barcode]; exists {
		return fmt.Errorf("duplicate barcode")
	}
	p.ID = len(m.products) + 1
	m.products[p.Barcode] = p
	return nil
}
func (m *mockProductRepo) GetProducts(ctx context.Context, q ProductsQuery) ([]Product, int, error) {
	if m.fail {
		return nil, 0, fmt.Errorf("db error")
	}
	var result []Product
	for _, p := range m.products {
		result = append(result, *p)
	}
	return result, len(result), nil
}
func (m *mockProductRepo) GetProductByBarcode(ctx context.Context, barcode string) (*Product, error) {
	if m.fail {
		return nil, fmt.Errorf("db error")
	}
	p, ok := m.products[barcode]
	if !ok {
		return nil, nil
	}
	return p, nil
}
func (m *mockProductRepo) UpdateProduct(ctx context.Context, id int, p *Product) error {
	if m.fail {
		return fmt.Errorf("db error")
	}
	for _, prod := range m.products {
		if prod.ID == id {
			*prod = *p
			prod.ID = id
			return nil
		}
	}
	return fmt.Errorf("not found")
}
func (m *mockProductRepo) DeleteProduct(ctx context.Context, id int) error {
	if m.fail {
		return fmt.Errorf("db error")
	}
	for k, prod := range m.products {
		if prod.ID == id {
			delete(m.products, k)
			return nil
		}
	}
	return fmt.Errorf("not found")
}
func (m *mockProductRepo) StockEntry(ctx context.Context, barcode string, qty int) error {
	if m.fail {
		return fmt.Errorf("db error")
	}
	p, ok := m.products[barcode]
	if !ok {
		return fmt.Errorf("not found")
	}
	p.Quantity += qty
	return nil
}
func (m *mockProductRepo) StockExit(ctx context.Context, barcode string, qty int) error {
	if m.fail {
		return fmt.Errorf("db error")
	}
	p, ok := m.products[barcode]
	if !ok || p.Quantity < qty {
		return fmt.Errorf("insufficient stock or not found")
	}
	p.Quantity -= qty
	return nil
}

func TestService_CreateProduct_Mock(t *testing.T) {
	repo := &mockProductRepo{products: make(map[string]*Product)}
	svc := NewService(repo)
	p := &Product{Name: "Produto Teste", Barcode: "123", Quantity: 10, MinStock: 2}
	err := svc.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatalf("erro ao criar produto: %v", err)
	}
	if p.ID == 0 {
		t.Error("ID não foi preenchido")
	}
	// Duplicidade
	err = svc.CreateProduct(context.Background(), &Product{Name: "Produto Teste", Barcode: "123", Quantity: 1, MinStock: 1})
	if err == nil {
		t.Error("esperado erro de duplicidade")
	}
}

func TestService_GetProductByBarcode_Mock(t *testing.T) {
	repo := &mockProductRepo{products: make(map[string]*Product)}
	svc := NewService(repo)
	p := &Product{Name: "Produto Teste", Barcode: "123", Quantity: 10, MinStock: 2}
	_ = svc.CreateProduct(context.Background(), p)
	prod, err := svc.GetProductByBarcode(context.Background(), "123")
	if err != nil || prod == nil {
		t.Fatalf("erro ao buscar produto: %v", err)
	}
	if prod.Barcode != "123" {
		t.Errorf("barcode incorreto: %v", prod.Barcode)
	}
	// Não encontrado
	prod, err = svc.GetProductByBarcode(context.Background(), "999")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if prod != nil {
		t.Error("esperado nil para produto inexistente")
	}
}

func TestService_StockEntryExit_Mock(t *testing.T) {
	repo := &mockProductRepo{products: make(map[string]*Product)}
	svc := NewService(repo)
	p := &Product{Name: "Produto Teste", Barcode: "123", Quantity: 10, MinStock: 2}
	_ = svc.CreateProduct(context.Background(), p)
	err := svc.StockEntry(context.Background(), "123", 5)
	if err != nil {
		t.Fatalf("erro ao dar entrada: %v", err)
	}
	if p.Quantity != 15 {
		t.Errorf("esperado 15, veio %d", p.Quantity)
	}
	err = svc.StockExit(context.Background(), "123", 10)
	if err != nil {
		t.Fatalf("erro ao dar saída: %v", err)
	}
	if p.Quantity != 5 {
		t.Errorf("esperado 5, veio %d", p.Quantity)
	}
	// Estoque insuficiente
	err = svc.StockExit(context.Background(), "123", 99)
	if err == nil {
		t.Error("esperado erro de estoque insuficiente")
	}
}

func TestService_Failures_Mock(t *testing.T) {
	repo := &mockProductRepo{products: make(map[string]*Product), fail: true}
	svc := NewService(repo)
	p := &Product{Name: "Produto Teste", Barcode: "123", Quantity: 10, MinStock: 2}
	if err := svc.CreateProduct(context.Background(), p); err == nil {
		t.Error("esperado erro de banco")
	}
	if _, _, err := svc.GetProducts(context.Background(), ProductsQuery{}); err == nil {
		t.Error("esperado erro de banco")
	}
	if _, err := svc.GetProductByBarcode(context.Background(), "123"); err == nil {
		t.Error("esperado erro de banco")
	}
	if err := svc.UpdateProduct(context.Background(), 1, p); err == nil {
		t.Error("esperado erro de banco")
	}
	if err := svc.DeleteProduct(context.Background(), 1); err == nil {
		t.Error("esperado erro de banco")
	}
	if err := svc.StockEntry(context.Background(), "123", 1); err == nil {
		t.Error("esperado erro de banco")
	}
	if err := svc.StockExit(context.Background(), "123", 1); err == nil {
		t.Error("esperado erro de banco")
	}
}
