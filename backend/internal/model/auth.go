package model

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse carries the session tokens back to the handler, which writes
// them to httpOnly cookies. The tokens are never serialized into the JSON
// response body.
type LoginResponse struct {
	AccessToken string `json:"-"`
	Username    string `json:"username"`
}

type PageInfo struct {
	CurrentPage     int  `json:"current_page"`
	Limit           int  `json:"limit"`
	Total           int  `json:"total"`
	TotalPages      int  `json:"total_pages"`
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
}
