package identity

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type storeFake struct {
	user    *User
	findErr error
	created *User
}

func (f *storeFake) Create(_ context.Context, user *User) error {
	f.created = user
	return nil
}

func (f *storeFake) List(context.Context) ([]*User, error) { return nil, nil }
func (f *storeFake) FindByEmail(context.Context, string) (*User, error) {
	return f.user, f.findErr
}
func (f *storeFake) UpdatePassword(context.Context, string, string) error { return nil }
func (f *storeFake) Delete(context.Context, string) error                 { return nil }

type tokenIssuerFake struct {
	token string
	err   error
}

func (f tokenIssuerFake) Generate(string, string) (string, error) {
	return f.token, f.err
}

func TestRegisterHashesPassword(t *testing.T) {
	store := &storeFake{findErr: errors.New("not found")}
	service := NewService(store, tokenIssuerFake{})

	if err := service.Register(context.Background(), "user@example.com", "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if store.created == nil {
		t.Fatal("Register() did not create a user")
	}
	if store.created.PasswordHash == "secret" {
		t.Fatal("Register() stored the plain-text password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.created.PasswordHash), []byte("secret")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}
}

func TestRegisterReportsExistingUser(t *testing.T) {
	service := NewService(&storeFake{user: &User{Email: "user@example.com"}}, tokenIssuerFake{})

	if err := service.Register(context.Background(), "user@example.com", "secret"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("Register() error = %v, want %v", err, ErrUserExists)
	}
}

func TestLoginReturnsTokenAndUser(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}
	user := &User{ID: "user-id", Email: "user@example.com", PasswordHash: string(hash)}
	service := NewService(&storeFake{user: user}, tokenIssuerFake{token: "signed-token"})

	result, err := service.Login(context.Background(), user.Email, "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token != "signed-token" || result.User.ID != user.ID {
		t.Fatalf("Login() result = %#v", result)
	}
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}
	service := NewService(
		&storeFake{user: &User{PasswordHash: string(hash)}},
		tokenIssuerFake{token: "signed-token"},
	)

	if _, err := service.Login(context.Background(), "user@example.com", "wrong"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Login() error = %v, want %v", err, ErrUnauthorized)
	}
}
