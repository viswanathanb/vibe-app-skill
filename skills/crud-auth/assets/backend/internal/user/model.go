package user

import (
	"time"

	"example.com/app/internal/authz"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"size:320;not null;uniqueIndex" json:"email"` // always lower-case
	Name         string    `gorm:"size:200;not null;default:''" json:"name"`
	PasswordHash string    `gorm:"size:100;not null;default:''" json:"-"` // empty for SSO-only users
	Role         string    `gorm:"size:20;not null;default:member" json:"role"`
	Disabled     bool      `gorm:"not null;default:false" json:"disabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (u *User) Principal() authz.Principal { return authz.Principal{UserID: u.ID, Role: u.Role} }

// CreateInput is used by admins to create users (production default is invite-only).
type CreateInput struct {
	Email    string `json:"email" binding:"required,email,max=320"`
	Name     string `json:"name" binding:"max=200"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Role     string `json:"role" binding:"required,oneof=admin member viewer"`
}

type UpdateInput struct {
	Name     *string `json:"name" binding:"omitempty,max=200"`
	Role     *string `json:"role" binding:"omitempty,oneof=admin member viewer"`
	Disabled *bool   `json:"disabled"`
}
