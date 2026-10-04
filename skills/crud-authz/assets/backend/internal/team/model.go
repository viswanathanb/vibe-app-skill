package team

import (
	"time"

	"example.com/app/internal/authz"
)

// ObjectType is the authz type name for teams (see authz.AppSchema).
const ObjectType = "team"

// ActionCreate is the RBAC action required to create teams (granted in authz.RoleActions).
const ActionCreate = ObjectType + ":create"

// Team is a group of users. Membership is stored as authz tuples (team:<id>#member@user:<id>),
// so projects and other resources can be shared with a whole team (team:<id>#member).
type Team struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"size:2000;not null;default:''" json:"description"`
	CreatedByID uint      `gorm:"not null;index" json:"createdById"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (t *Team) Object() authz.Object { return authz.Obj(ObjectType, t.ID) }

type CreateInput struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

type UpdateInput struct {
	Name        *string `json:"name" binding:"omitempty,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
}

// Detail is the single-item response: the team plus the caller's permissions on it.
type Detail struct {
	Team
	Permissions []string `json:"permissions"`
}
