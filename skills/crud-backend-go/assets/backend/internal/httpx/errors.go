package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// Error is the single error type returned to API clients:
// {"error": {"code": "not_found", "message": "...", "details": {...}}}
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(msg string) *Error   { return NewError(http.StatusBadRequest, "bad_request", msg) }
func Unauthorized(msg string) *Error { return NewError(http.StatusUnauthorized, "unauthorized", msg) }
func Forbidden(msg string) *Error    { return NewError(http.StatusForbidden, "forbidden", msg) }
func NotFound(msg string) *Error     { return NewError(http.StatusNotFound, "not_found", msg) }
func Conflict(msg string) *Error     { return NewError(http.StatusConflict, "conflict", msg) }
func TooManyRequests(msg string) *Error {
	return NewError(http.StatusTooManyRequests, "too_many_requests", msg)
}

// Fail maps err to a JSON error response and aborts the request.
// Unknown errors are logged and returned as a generic 500 so internals never leak.
func Fail(c *gin.Context, err error) {
	var apiErr *Error
	switch {
	case errors.As(err, &apiErr):
	case errors.Is(err, gorm.ErrRecordNotFound):
		apiErr = NotFound("resource not found")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		apiErr = Conflict("resource already exists")
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		apiErr = Conflict("resource is referenced by or references another resource")
	default:
		slog.ErrorContext(c.Request.Context(), "request failed",
			"err", err, "method", c.Request.Method, "route", c.FullPath())
		apiErr = NewError(http.StatusInternalServerError, "internal", "internal server error")
	}
	c.AbortWithStatusJSON(apiErr.Status, gin.H{"error": apiErr})
}

// BindJSON decodes and validates (via `binding` struct tags) the request body.
// On failure it writes a 400 with per-field details and returns false.
func BindJSON(c *gin.Context, dst any) bool {
	err := c.ShouldBindJSON(dst)
	if err == nil {
		return true
	}
	apiErr := BadRequest("invalid request body")
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			msg := fe.Tag()
			if fe.Param() != "" {
				msg += "=" + fe.Param()
			}
			fields[fe.Field()] = msg
		}
		apiErr.Message = "validation failed"
		apiErr.Details = fields
	}
	Fail(c, apiErr)
	return false
}

// ParseID reads a positive integer path parameter.
func ParseID(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || id == 0 {
		Fail(c, BadRequest("invalid "+name))
		return 0, false
	}
	return uint(id), true
}

// UseJSONFieldNames makes validation errors report `json` tag names instead of Go field names.
func UseJSONFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" || name == "" {
			return f.Name
		}
		return name
	})
}
