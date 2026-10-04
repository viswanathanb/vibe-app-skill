package team

import (
	"context"

	"gorm.io/gorm"

	"example.com/app/internal/httpx"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

var sortColumns = map[string]string{"name": "name", "createdAt": "created_at", "updatedAt": "updated_at"}

type ListFilter struct {
	Page httpx.Page
	IDs  []uint // restrict to these IDs unless All
	All  bool
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]Team, int64, error) {
	items := []Team{}
	if !f.All && len(f.IDs) == 0 {
		return items, 0, nil
	}
	q := r.db.WithContext(ctx).Model(&Team{})
	if !f.All {
		q = q.Where("id IN ?", f.IDs)
	}
	if f.Page.Query != "" {
		q = q.Where("name ILIKE ?", httpx.LikePattern(f.Page.Query))
	}
	q = q.Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order(httpx.OrderBy(f.Page.Sort, sortColumns, "name ASC")).
		Limit(f.Page.Limit).Offset(f.Page.Offset).Find(&items).Error
	return items, total, err
}

func (r *Repository) Get(ctx context.Context, id uint) (*Team, error) {
	var t Team
	if err := r.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) Create(ctx context.Context, t *Team) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *Repository) Save(ctx context.Context, t *Team) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&Team{}, id).Error
}
