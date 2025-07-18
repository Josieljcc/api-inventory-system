package users

import "time"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type RefreshToken struct {
	ID        int
	UserID    int
	Token     string
	ExpiresAt time.Time
}
