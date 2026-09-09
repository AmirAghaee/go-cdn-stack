package httpapi

import (
	"time"

	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
)

type credentialsRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type passwordRequest struct {
	Password string `json:"password" binding:"required"`
}

func newUserResponse(user *identity.User) userResponse {
	return userResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt}
}
