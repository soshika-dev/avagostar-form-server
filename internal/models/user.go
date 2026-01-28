package models

import "time"

type User struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Role               string     `json:"role"`
	PasswordHash       string     `json:"-"`
	ResetCodeHash      *string    `json:"-"`
	ResetCodeExpiresAt *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
