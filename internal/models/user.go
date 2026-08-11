package models

import "time"

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleServiceDesk Role = "service_desk"
	RoleEngineer    Role = "engineer"
	RoleRecruiter   Role = "recruiter"
	RoleAccounts    Role = "accounts"
)

func IsValidRole(r Role) bool {
	switch r {
	case RoleAdmin, RoleServiceDesk, RoleEngineer, RoleRecruiter, RoleAccounts:
		return true
	}
	return false
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"` // never serialized to API responses
	CreatedAt    time.Time `json:"created_at"`
}

type NewUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}
