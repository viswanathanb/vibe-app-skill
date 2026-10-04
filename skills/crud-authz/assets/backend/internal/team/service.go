package team

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
)

type Service struct {
	db   *gorm.DB
	repo *Repository
	az   *authz.Engine
}

func NewService(db *gorm.DB, repo *Repository, az *authz.Engine) *Service {
	return &Service{db: db, repo: repo, az: az}
}

func (s *Service) List(ctx context.Context, p authz.Principal, page httpx.Page) (httpx.List[Team], error) {
	ids, all, err := s.az.LookupResources(ctx, p, "view", ObjectType)
	if err != nil {
		return httpx.List[Team]{}, err
	}
	items, total, err := s.repo.List(ctx, ListFilter{Page: page, IDs: authz.UintIDs(ids), All: all})
	if err != nil {
		return httpx.List[Team]{}, err
	}
	return httpx.List[Team]{Items: items, Total: total, Limit: page.Limit, Offset: page.Offset}, nil
}

func (s *Service) Get(ctx context.Context, p authz.Principal, id uint) (*Detail, error) {
	obj := authz.Obj(ObjectType, id)
	if err := s.az.Authorize(ctx, p, "view", obj); err != nil {
		return nil, err
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	perms, err := s.az.Permissions(ctx, p, obj)
	if err != nil {
		return nil, err
	}
	return &Detail{Team: *t, Permissions: perms}, nil
}

func (s *Service) Create(ctx context.Context, p authz.Principal, in CreateInput) (*Team, error) {
	if !p.Can(ActionCreate) {
		return nil, httpx.Forbidden("your role cannot create teams")
	}
	t := &Team{Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), CreatedByID: p.UserID}
	if t.Name == "" {
		return nil, httpx.BadRequest("name is required")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTx(tx).Create(ctx, t); err != nil {
			return err
		}
		// The creator administers the team and is also a member (so team shares apply to them).
		return s.az.WithTx(tx).Grant(ctx,
			authz.NewTuple(t.Object(), "admin", p.Subject()),
			authz.NewTuple(t.Object(), "member", p.Subject()),
		)
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, p authz.Principal, id uint, in UpdateInput) (*Team, error) {
	if err := s.az.Authorize(ctx, p, "edit", authz.Obj(ObjectType, id)); err != nil {
		return nil, err
	}
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		t.Name = strings.TrimSpace(*in.Name)
		if t.Name == "" {
			return nil, httpx.BadRequest("name is required")
		}
	}
	if in.Description != nil {
		t.Description = strings.TrimSpace(*in.Description)
	}
	if err := s.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
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
