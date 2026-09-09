package jwttoken

import (
	"github.com/AmirAghaee/go-cdn-stack/control-panel/internal/identity"
	"github.com/AmirAghaee/go-cdn-stack/pkg/jwt"
)

type Manager struct {
	manager *jwt.Manager
}

func New(manager *jwt.Manager) *Manager {
	return &Manager{manager: manager}
}

func (m *Manager) Generate(userID, email string) (string, error) {
	return m.manager.Generate(userID, email)
}

func (m *Manager) Verify(token string) (identity.Claims, error) {
	claims, err := m.manager.Verify(token)
	if err != nil {
		return identity.Claims{}, err
	}
	return identity.Claims{UserID: claims.UserID, Email: claims.Email}, nil
}
