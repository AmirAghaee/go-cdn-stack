package identity

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists      = errors.New("user already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrSelfDelete      = errors.New("you cannot delete your own account")
	ErrUnauthorized    = errors.New("invalid credentials")
)

type Store interface {
	Create(context.Context, *User) error
	List(context.Context) ([]*User, error)
	FindByEmail(context.Context, string) (*User, error)
	UpdatePassword(context.Context, string, string) error
	Delete(context.Context, string) error
}

type TokenIssuer interface {
	Generate(userID, email string) (string, error)
}

type TokenVerifier interface {
	Verify(string) (Claims, error)
}

type LoginResult struct {
	Token string
	User  User
}

type Service struct {
	store  Store
	tokens TokenIssuer
}

func NewService(store Store, tokens TokenIssuer) *Service {
	return &Service{store: store, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, email, password string) error {
	if user, _ := s.store.FindByEmail(ctx, email); user != nil {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.Create(ctx, &User{Email: email, PasswordHash: string(hash)})
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.store.FindByEmail(ctx, email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}

	token, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: *user}, nil
}

func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.store.List(ctx)
}

func (s *Service) ChangePassword(ctx context.Context, userID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, userID, string(hash))
}

func (s *Service) Delete(ctx context.Context, actorID, userID string) error {
	if actorID == userID {
		return ErrSelfDelete
	}
	return s.store.Delete(ctx, userID)
}
