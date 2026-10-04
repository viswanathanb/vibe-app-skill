package project

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
)

// Controller is the HTTP layer only: parse input, call the service, write JSON.
type Controller struct{ svc *Service }

func NewController(svc *Service) *Controller { return &Controller{svc: svc} }

func (ctl *Controller) Register(rg *gin.RouterGroup) {
	g := rg.Group("/projects")
	g.GET("", ctl.list)
	g.POST("", ctl.create)
	g.GET("/:id", ctl.get)
	g.PATCH("/:id", ctl.update)
	g.DELETE("/:id", ctl.remove)
}

// list: GET /api/projects?limit=&offset=&sort=-updatedAt&q=&status=
func (ctl *Controller) list(c *gin.Context) {
	res, err := ctl.svc.List(c.Request.Context(), authz.PrincipalFrom(c), httpx.ParsePage(c), c.Query("status"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (ctl *Controller) get(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	res, err := ctl.svc.Get(c.Request.Context(), authz.PrincipalFrom(c), id)
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

func (ctl *Controller) remove(c *gin.Context) {
	id, ok := httpx.ParseID(c, "id")
	if !ok {
		return
	}
	if err := ctl.svc.Delete(c.Request.Context(), authz.PrincipalFrom(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
