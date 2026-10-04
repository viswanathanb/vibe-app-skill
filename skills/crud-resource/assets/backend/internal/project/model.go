package project

import (
	"time"

	"example.com/app/internal/authz"
)

// ObjectType is the authz type name (must exist in authz.AppSchema).
const ObjectType = "project"

// ActionCreate is the RBAC action required to create projects (granted in authz.RoleActions).
const ActionCreate = ObjectType + ":create"

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"size:2000;not null;default:''" json:"description"`
	Status      string    `gorm:"size:20;not null;default:active;index" json:"status"`
	CreatedByID uint      `gorm:"not null;index" json:"createdById"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (p *Project) Object() authz.Object { return authz.Obj(ObjectType, p.ID) }

type CreateInput struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

// UpdateInput uses pointers so omitted fields are left unchanged (PATCH semantics).
type UpdateInput struct {
	Name        *string `json:"name" binding:"omitempty,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Status      *string `json:"status" binding:"omitempty,oneof=active archived"`
}

// Detail is the single-item response: the project plus the caller's permissions on it.
type Detail struct {
	Project
	Permissions []string `json:"permissions"`
}
