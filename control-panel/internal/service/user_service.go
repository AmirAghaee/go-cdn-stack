package service

import (
	"context"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/helper"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/repository"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserServiceInterface interface {
	Register(ctx context.Context, email, password string) error
	Login(ctx context.Context, email, password string) (*LoginResponse, error)
	List(ctx context.Context) ([]*domain.User, error)
}

type UserService struct {
	repo       repository.UserRepositoryInterface
	jwtManager *jwt.Manager
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func NewUserService(repo repository.UserRepositoryInterface, jwtManager *jwt.Manager) *UserService {
	return &UserService{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

func (s *UserService) Register(ctx context.Context, email, password string) error {
	existing, _ := s.repo.FindByEmail(ctx, email)
	if existing != nil {
		return helper.ErrUserExists()
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		Email:    email,
		Password: string(hashedPassword),
	}

	return s.repo.Create(ctx, user)
}

func (s *UserService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, helper.ErrUnAuthorized()
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, helper.ErrUnAuthorized()
	}

	// Generate JWT token
	token, err := s.jwtManager.Generate(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *UserService) List(ctx context.Context) ([]*domain.User, error) {
	return s.repo.FindAll(ctx)
}
