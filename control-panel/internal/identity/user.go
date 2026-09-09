package identity

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Claims struct {
	UserID string
	Email  string
}
