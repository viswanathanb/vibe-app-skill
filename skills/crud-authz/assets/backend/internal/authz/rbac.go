package authz

import (
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"

	"example.com/app/internal/httpx"
)

// Global roles (RBAC). Stored on users.role.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Roles is the list of valid roles, used for validation.
var Roles = []string{RoleAdmin, RoleMember, RoleViewer}

// Actions are app-wide capabilities granted by global role. They answer "may this user do X at all?"
// (e.g. create a project). Per-object access ("may this user edit project 42?") is ReBAC: see app_schema.go.
// Resource packages define their own `ActionCreate = "<type>:create"` and reference it here as a string.
const ActionManageUsers = "users:manage"

// RoleActions maps each role to its actions. "*" grants everything.
var RoleActions = map[string][]string{
	RoleAdmin:  {"*"},
	RoleMember: {"team:create" /* crud:actions */},
	RoleViewer: {},
}

// Principal is the authenticated caller. The auth middleware stores it on the gin context.
type Principal struct {
	UserID uint
	Role   string
}

func (p Principal) IsAdmin() bool { return p.Role == RoleAdmin }

func (p Principal) Subject() Subject {
	return Subject{Type: TypeUser, ID: strconv.FormatUint(uint64(p.UserID), 10)}
}

// Can reports whether the principal's global role grants action.
func (p Principal) Can(action string) bool {
	acts := RoleActions[p.Role]
	return slices.Contains(acts, "*") || slices.Contains(acts, action)
}

// ActionsFor returns the role's actions for the UI (it hides buttons the user cannot use).
func ActionsFor(role string) []string {
	acts := RoleActions[role]
	if acts == nil {
		return []string{}
	}
	return acts
}

const principalKey = "authz.principal"

func SetPrincipal(c *gin.Context, p Principal) { c.Set(principalKey, p) }

// PrincipalFrom returns the caller, or a zero Principal (no access to anything) if unauthenticated.
func PrincipalFrom(c *gin.Context) Principal {
	p, _ := c.Get(principalKey)
	pr, _ := p.(Principal)
	return pr
}

// RequireAction is route middleware for role-gated endpoints (e.g. admin-only user management).
func RequireAction(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !PrincipalFrom(c).Can(action) {
			httpx.Fail(c, httpx.Forbidden("your role does not allow "+action))
			return
		}
		c.Next()
	}
}
