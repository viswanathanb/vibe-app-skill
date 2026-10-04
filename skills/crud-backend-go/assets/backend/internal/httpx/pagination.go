package httpx

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Page holds list query parameters: ?limit=20&offset=0&sort=-createdAt&q=text
type Page struct {
	Limit  int
	Offset int
	Sort   string
	Query  string
}

// List is the response envelope for every list endpoint.
type List[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

func ParsePage(c *gin.Context) Page {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)
	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}
	return Page{
		Limit:  limit,
		Offset: offset,
		Sort:   c.Query("sort"),
		Query:  strings.TrimSpace(c.Query("q")),
	}
}

// OrderBy turns a client sort key ("name" or "-createdAt") into an ORDER BY clause.
// Only keys present in allowed (API key -> column) are accepted, which prevents SQL injection.
func OrderBy(sort string, allowed map[string]string, fallback string) string {
	desc := strings.HasPrefix(sort, "-")
	col, ok := allowed[strings.TrimPrefix(sort, "-")]
	if !ok {
		return fallback
	}
	if desc {
		return col + " DESC"
	}
	return col + " ASC"
}

// LikePattern escapes LIKE wildcards in user input and wraps it for a contains-match.
func LikePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}
