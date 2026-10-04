package project

import (
	"context"

	"gorm.io/gorm"

	"example.com/app/internal/httpx"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// WithTx returns a repository bound to a transaction.
func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

// sortColumns is the allow-list of client sort keys -> SQL columns.
var sortColumns = map[string]string{
	"name":      "name",
	"status":    "status",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}

type ListFilter struct {
	Page   httpx.Page
	IDs    []uint // restrict to these IDs unless All (from authz.LookupResources)
	All    bool
	Status string
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]Project, int64, error) {
	items := []Project{}
	if !f.All && len(f.IDs) == 0 {
		return items, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&Project{})
	if !f.All {
		q = q.Where("id IN ?", f.IDs)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Page.Query != "" {
		like := httpx.LikePattern(f.Page.Query)
		q = q.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	q = q.Session(&gorm.Session{}) // make q reusable for Count + Find

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order(httpx.OrderBy(f.Page.Sort, sortColumns, "updated_at DESC")).
		Limit(f.Page.Limit).Offset(f.Page.Offset).Find(&items).Error
	return items, total, err
}

func (r *Repository) Get(ctx context.Context, id uint) (*Project, error) {
	var p Project
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repository) Save(ctx context.Context, p *Project) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Project{}, id).Error
}
