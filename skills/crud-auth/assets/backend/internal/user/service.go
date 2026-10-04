package user

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
)

// Service implements admin user management. Every method requires users:manage.
type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, p authz.Principal, page httpx.Page) (httpx.List[User], error) {
	if !p.Can(authz.ActionManageUsers) {
		return httpx.List[User]{}, httpx.Forbidden("admins only")
	}
	items, total, err := s.repo.List(ctx, page)
	if err != nil {
		return httpx.List[User]{}, err
	}
	return httpx.List[User]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (s *Service) Create(ctx context.Context, p authz.Principal, in CreateInput) (*User, error) {
	if !p.Can(authz.ActionManageUsers) {
		return nil, httpx.Forbidden("admins only")
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := &User{Email: in.Email, Name: strings.TrimSpace(in.Name), PasswordHash: hash, Role: in.Role}
	if err := s.repo.Create(ctx, u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, httpx.Conflict("a user with this email already exists")
		}
		return nil, err
	}
	return u, nil
}

func (s *Service) Update(ctx context.Context, p authz.Principal, id uint, in UpdateInput) (*User, error) {
	if !p.Can(authz.ActionManageUsers) {
		return nil, httpx.Forbidden("admins only")
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	losesAdmin := u.Role == authz.RoleAdmin && !u.Disabled &&
		((in.Role != nil && *in.Role != authz.RoleAdmin) || (in.Disabled != nil && *in.Disabled))
	if losesAdmin {
		n, err := s.repo.CountActiveAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, httpx.Conflict("cannot demote or disable the last admin")
		}
	}
	if in.Name != nil {
		u.Name = strings.TrimSpace(*in.Name)
	}
	if in.Role != nil {
		u.Role = *in.Role
	}
	if in.Disabled != nil {
		u.Disabled = *in.Disabled
	}
	if err := s.repo.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
