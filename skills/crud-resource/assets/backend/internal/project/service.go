package project

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
)

// Service holds business rules. Every method takes the caller's Principal and authorizes FIRST:
// RBAC (p.Can) for "may create at all", ReBAC (az.Authorize / LookupResources) for specific objects.
type Service struct {
	db   *gorm.DB // only used to open transactions
	repo *Repository
	az   *authz.Engine
}

func NewService(db *gorm.DB, repo *Repository, az *authz.Engine) *Service {
	return &Service{db: db, repo: repo, az: az}
}

func (s *Service) List(ctx context.Context, p authz.Principal, page httpx.Page, status string) (httpx.List[Project], error) {
	ids, all, err := s.az.LookupResources(ctx, p, "view", ObjectType)
	if err != nil {
		return httpx.List[Project]{}, err
	}
	items, total, err := s.repo.List(ctx, ListFilter{Page: page, IDs: authz.UintIDs(ids), All: all, Status: status})
	if err != nil {
		return httpx.List[Project]{}, err
	}
	return httpx.List[Project]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (s *Service) Get(ctx context.Context, p authz.Principal, id uint) (*Detail, error) {
	obj := authz.Obj(ObjectType, id)
	if err := s.az.Authorize(ctx, p, "view", obj); err != nil {
		return nil, err
	}
	proj, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	perms, err := s.az.Permissions(ctx, p, obj)
	if err != nil {
		return nil, err
	}
	return &Detail{Project: *proj, Permissions: perms}, nil
}

func (s *Service) Create(ctx context.Context, p authz.Principal, in CreateInput) (*Project, error) {
	if !p.Can(ActionCreate) {
		return nil, httpx.Forbidden("your role cannot create projects")
	}
	proj := &Project{
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Status:      StatusActive,
		CreatedByID: p.UserID,
	}
	if proj.Name == "" {
		return nil, httpx.BadRequest("name is required")
	}
	// Row + owner tuple are written atomically so a project never exists without an owner.
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTx(tx).Create(ctx, proj); err != nil {
			return err
		}
		return s.az.WithTx(tx).Grant(ctx, authz.NewTuple(proj.Object(), "owner", p.Subject()))
	})
	if err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *Service) Update(ctx context.Context, p authz.Principal, id uint, in UpdateInput) (*Project, error) {
	if err := s.az.Authorize(ctx, p, "edit", authz.Obj(ObjectType, id)); err != nil {
		return nil, err
	}
	proj, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		proj.Name = strings.TrimSpace(*in.Name)
		if proj.Name == "" {
			return nil, httpx.BadRequest("name is required")
		}
	}
	if in.Description != nil {
		proj.Description = strings.TrimSpace(*in.Description)
	}
	if in.Status != nil {
		proj.Status = *in.Status
	}
	if err := s.repo.Save(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *Service) Delete(ctx context.Context, p authz.Principal, id uint) error {
	obj := authz.Obj(ObjectType, id)
	if err := s.az.Authorize(ctx, p, "delete", obj); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTx(tx).Delete(ctx, id); err != nil {
			return err
		}
		return s.az.WithTx(tx).DeleteObject(ctx, obj)
	})
}
