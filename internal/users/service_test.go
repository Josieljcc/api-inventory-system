package users_test

import (
	"context"
	"errors"
	"inventory-system/internal/users"
	"testing"
	"time"
)

type MockUserRepo struct {
	users map[string]*users.User
	fail  bool
}

func (m *MockUserRepo) CreateUser(ctx context.Context, username, passwordHash, role string) error {
	if m.fail {
		return errors.New("db error")
	}
	if _, exists := m.users[username]; exists {
		return errors.New("duplicate username")
	}
	m.users[username] = &users.User{ID: len(m.users) + 1, Username: username, PasswordHash: passwordHash, Role: role}
	return nil
}

func (m *MockUserRepo) GetUserByUsername(ctx context.Context, username string) (*users.User, error) {
	if m.fail {
		return nil, errors.New("db error")
	}
	u, ok := m.users[username]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *MockUserRepo) CreateRefreshToken(ctx context.Context, userID int, token string, expiresAt time.Time) error {
	return nil
}

func (m *MockUserRepo) GetRefreshToken(ctx context.Context, token string) (*users.RefreshToken, error) {
	return nil, errors.New("not implemented")
}

func (m *MockUserRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	return nil
}

func TestService_RegisterAndAuth_Mock(t *testing.T) {
	repo := &MockUserRepo{users: make(map[string]*users.User)}
	svc := users.NewService(repo)
	// Registro
	err := svc.Register(context.Background(), "testuser", "testpass", "user")
	if err != nil {
		t.Fatalf("erro ao registrar: %v", err)
	}
	// Registro duplicado
	err = svc.Register(context.Background(), "testuser", "testpass", "user")
	if err == nil {
		t.Error("esperado erro de duplicidade")
	}
	// Autenticação correta
	user, err := svc.Authenticate(context.Background(), "testuser", "testpass")
	if err != nil || user == nil {
		t.Fatalf("erro ao autenticar: %v", err)
	}
	// Autenticação com senha errada
	_, err = svc.Authenticate(context.Background(), "testuser", "wrongpass")
	if err == nil {
		t.Error("esperado erro de senha inválida")
	}
	// Autenticação de usuário inexistente
	_, err = svc.Authenticate(context.Background(), "inexistent", "testpass")
	if err == nil {
		t.Error("esperado erro de usuário inexistente")
	}
}

func TestService_Failures_Mock(t *testing.T) {
	repo := &MockUserRepo{users: make(map[string]*users.User), fail: true}
	svc := users.NewService(repo)
	if err := svc.Register(context.Background(), "testuser", "testpass", "user"); err == nil {
		t.Error("esperado erro de banco no registro")
	}
	if _, err := svc.Authenticate(context.Background(), "testuser", "testpass"); err == nil {
		t.Error("esperado erro de banco na autenticação")
	}
	if _, err := svc.GetByUsername(context.Background(), "testuser"); err == nil {
		t.Error("esperado erro de banco na busca por username")
	}
}
