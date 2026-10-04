package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
)

type Controller struct{ svc *Service }

func NewController(svc *Service) *Controller { return &Controller{svc: svc} }

// Register mounts admin-only user management under /users.
func (ctl *Controller) Register(rg *gin.RouterGroup) {
	g := rg.Group("/users", authz.RequireAction(authz.ActionManageUsers))
	g.GET("", ctl.list)
	g.POST("", ctl.create)
	g.PATCH("/:id", ctl.update)
}

func (ctl *Controller) list(c *gin.Context) {
	res, err := ctl.svc.List(c.Request.Context(), authz.PrincipalFrom(c), httpx.ParsePage(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (ctl *Controller) create(c *gin.Context) {
	var in CreateInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	res, err := ctl.svc.Create(c.Request.Context(), authz.PrincipalFrom(c), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (ctl *Controller) update(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	var in UpdateInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	res, err := ctl.svc.Update(c.Request.Context(), authz.PrincipalFrom(c), id, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
