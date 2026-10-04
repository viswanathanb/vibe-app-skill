package user

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"example.com/app/internal/httpx"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (r *Repository) Create(ctx context.Context, u *User) error {
	u.Email = NormalizeEmail(u.Email)
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repository) FindByID(ctx context.Context, id uint) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("email = ?", NormalizeEmail(email)).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&User{}).Count(&n).Error
	return n, err
}

var sortColumns = map[string]string{"email": "email", "name": "name", "role": "role", "createdAt": "created_at"}

func (r *Repository) List(ctx context.Context, page httpx.Page) ([]User, int64, error) {
	q := r.db.WithContext(ctx).Model(&User{})
	if page.Query != "" {
		like := httpx.LikePattern(page.Query)
		q = q.Where("email ILIKE ? OR name ILIKE ?", like, like)
	}
	q = q.Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []User{}
	err := q.Order(httpx.OrderBy(page.Sort, sortColumns, "email ASC")).
		Limit(page.Limit).Offset(page.Offset).Find(&items).Error
	return items, total, err
}

func (r *Repository) Save(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *Repository) CountActiveAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("role = ? AND NOT disabled", "admin").Count(&n).Error
	return n, err
}

// LookupUserID implements authz.UserDirectory.
func (r *Repository) LookupUserID(ctx context.Context, email string) (uint, error) {
	u, err := r.FindByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}

// UserLabels implements authz.UserDirectory.
func (r *Repository) UserLabels(ctx context.Context, ids []uint) (map[uint]string, error) {
	out := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []User
	if err := r.db.WithContext(ctx).Select("id", "email", "name").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Name != "" {
			out[u.ID] = u.Name + " <" + u.Email + ">"
		} else {
			out[u.ID] = u.Email
		}
	}
	return out, nil
}
