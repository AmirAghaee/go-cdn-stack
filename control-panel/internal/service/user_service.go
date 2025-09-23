package service

import (
	"context"
	"errors"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/domain"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/helper"
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserServiceInterface interface {
	Register(ctx context.Context, email string, password string) error
	Login(ctx context.Context, email string, password string) (*domain.User, error)
	List(ctx context.Context) ([]*domain.User, error)
	Delete(ctx context.Context, id string) error
}

type UserService struct {
	repo repository.UserRepositoryInterface
}

func NewUserService(r repository.UserRepositoryInterface) UserServiceInterface {
	return &UserService{
		repo: r,
	}
}

func (u *UserService) Register(ctx context.Context, email string, password string) error {
	_, err := u.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return helper.ErrUserExists()
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &domain.User{Email: email, Password: string(hash)}
	return u.repo.CreateUser(ctx, user)
}

func (u *UserService) Login(ctx context.Context, email string, password string) (*domain.User, error) {
	user, err := u.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	user.Password = "" // hide password
	return user, nil
}

func (u *UserService) List(ctx context.Context) ([]*domain.User, error) {
	return u.repo.ListUser(ctx)
}

func (u *UserService) Delete(ctx context.Context, id string) error {
	return u.repo.DeleteUser(ctx, id)
}
