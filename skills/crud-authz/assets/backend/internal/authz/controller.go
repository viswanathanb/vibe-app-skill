package authz

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"example.com/app/internal/httpx"
)

// UserDirectory resolves users for the sharing API (implemented by user.Repository).
type UserDirectory interface {
	// LookupUserID returns gorm.ErrRecordNotFound when no user has that email.
	LookupUserID(ctx context.Context, email string) (uint, error)
	UserLabels(ctx context.Context, ids []uint) (map[uint]string, error)
}

// Controller exposes a generic sharing API for every schema type:
//
//	GET    /api/authz/:type/:id/permissions        caller's permissions on the object
//	GET    /api/authz/:type/:id/tuples              who has access (requires manage)
//	POST   /api/authz/:type/:id/tuples              grant  {relation, email | subject}
//	DELETE /api/authz/:type/:id/tuples?relation=&subject=   revoke
type Controller struct {
	engine *Engine
	users  UserDirectory
}

func NewController(engine *Engine, users UserDirectory) *Controller {
	return &Controller{engine: engine, users: users}
}

func (ctl *Controller) Register(rg *gin.RouterGroup) {
	g := rg.Group("/authz/:type/:id")
	g.GET("/permissions", ctl.permissions)
	g.GET("/tuples", ctl.list)
	g.POST("/tuples", ctl.grant)
	g.DELETE("/tuples", ctl.revoke)
}

type grantRequest struct {
	Relation string `json:"relation" binding:"required,max=64"`
	Email    string `json:"email" binding:"omitempty,email,max=320"`
	Subject  string `json:"subject" binding:"omitempty,max=200"`
}

type tupleView struct {
	Relation string `json:"relation"`
	Subject  string `json:"subject"`
	Label    string `json:"label"`
}

type tuplesResponse struct {
	Relations []string    `json:"relations"`
	Tuples    []tupleView `json:"tuples"`
}

func (ctl *Controller) object(c *gin.Context) (Object, bool) {
	obj := Object{Type: c.Param("type"), ID: c.Param("id")}
	if !ctl.engine.HasType(obj.Type) || !validID(obj.ID) {
		httpx.Fail(c, httpx.NotFound("unknown object"))
		return obj, false
	}
	return obj, true
}

func (ctl *Controller) permissions(c *gin.Context) {
	obj, ok := ctl.object(c)
	if !ok {
		return
	}
	perms, err := ctl.engine.Permissions(c.Request.Context(), PrincipalFrom(c), obj)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

func (ctl *Controller) list(c *gin.Context) {
	ctx := c.Request.Context()
	obj, ok := ctl.object(c)
	if !ok {
		return
	}
	if err := ctl.engine.Authorize(ctx, PrincipalFrom(c), "manage", obj); err != nil {
		httpx.Fail(c, err)
		return
	}
	tuples, err := ctl.engine.Tuples(ctx, obj)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	shareable := ctl.engine.ShareableRelations(obj.Type)

	var userIDs []uint
	for _, t := range tuples {
		if t.SubjectType == TypeUser {
			userIDs = append(userIDs, UintIDs([]string{t.SubjectID})...)
		}
	}
	labels, err := ctl.users.UserLabels(ctx, userIDs)
	if err != nil {
		httpx.Fail(c, err)
		return
	}

	views := make([]tupleView, 0, len(tuples))
	for _, t := range tuples {
		if !slices.Contains(shareable, t.Relation) {
			continue // structural links such as parent are not shown
		}
		label := t.Subject().String()
		if t.SubjectType == TypeUser {
			if id := UintIDs([]string{t.SubjectID}); len(id) == 1 && labels[id[0]] != "" {
				label = labels[id[0]]
			}
		}
		views = append(views, tupleView{Relation: t.Relation, Subject: t.Subject().String(), Label: label})
	}
	c.JSON(http.StatusOK, tuplesResponse{Relations: shareable, Tuples: views})
}

func (ctl *Controller) grant(c *gin.Context) {
	ctx := c.Request.Context()
	obj, ok := ctl.object(c)
	if !ok {
		return
	}
	var req grantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := ctl.engine.Authorize(ctx, PrincipalFrom(c), "manage", obj); err != nil {
		httpx.Fail(c, err)
		return
	}
	if !slices.Contains(ctl.engine.ShareableRelations(obj.Type), req.Relation) {
		httpx.Fail(c, httpx.BadRequest("relation cannot be shared"))
		return
	}
	sub, err := ctl.resolveSubject(ctx, req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := ctl.engine.Grant(ctx, NewTuple(obj, req.Relation, sub)); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *Controller) revoke(c *gin.Context) {
	ctx := c.Request.Context()
	obj, ok := ctl.object(c)
	if !ok {
		return
	}
	if err := ctl.engine.Authorize(ctx, PrincipalFrom(c), "manage", obj); err != nil {
		httpx.Fail(c, err)
		return
	}
	relation := c.Query("relation")
	sub, err := ParseSubject(c.Query("subject"))
	if err != nil || !slices.Contains(ctl.engine.ShareableRelations(obj.Type), relation) {
		httpx.Fail(c, httpx.BadRequest("invalid relation or subject"))
		return
	}
	if err := ctl.ensureNotLastManager(ctx, obj, relation, sub); err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := ctl.engine.Revoke(ctx, NewTuple(obj, relation, sub)); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ensureNotLastManager stops users from removing the last direct holder of a relation that
// grants `manage` (e.g. the only owner), which would orphan the object.
func (ctl *Controller) ensureNotLastManager(ctx context.Context, obj Object, relation string, sub Subject) error {
	t := ctl.engine.types[obj.Type]
	grantsManage := slices.ContainsFunc(t.permissions["manage"], func(tm term) bool {
		return tm.via == "" && tm.name == relation
	})
	if !grantsManage {
		return nil
	}
	holders, err := ctl.engine.Subjects(ctx, obj, relation)
	if err != nil {
		return err
	}
	if len(holders) == 1 && holders[0] == sub {
		return httpx.Conflict("cannot remove the last " + relation)
	}
	return nil
}

func (ctl *Controller) resolveSubject(ctx context.Context, req grantRequest) (Subject, error) {
	switch {
	case req.Email != "":
		id, err := ctl.users.LookupUserID(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Subject{}, httpx.NotFound("no user with that email")
		}
		if err != nil {
			return Subject{}, err
		}
		return Subject{Type: TypeUser, ID: strconv.FormatUint(uint64(id), 10)}, nil
	case req.Subject != "":
		sub, err := ParseSubject(req.Subject)
		if err != nil {
			return Subject{}, httpx.BadRequest(err.Error())
		}
		return sub, nil
	default:
		return Subject{}, httpx.BadRequest("email or subject is required")
	}
}
