# Access patterns

Every resource uses exactly one pattern. Write the choice into `APP_SPEC.md`.

## A. Owned + shareable (default)

Creator becomes `owner`; owner shares with users or teams as `editor`/`viewer`.

```go
// authz/app_schema.go
"document": {
	Relations: map[string][]string{
		"owner":  {"user"},
		"editor": {"user", "team#member"},
		"viewer": {"user", "team#member"},
	},
	Permissions: map[string][]string{
		"view":   {"viewer", "edit"},
		"edit":   {"editor", "manage"},
		"delete": {"owner"},
		"manage": {"owner"},
	},
},
```

Service: exactly like `crud-resource/assets/backend/internal/project/service.go`.

## B. Child of a parent (inherits access)

Tasks in a project, comments on a ticket, line items in an order. The child row has a `ParentID` column **and**
a `parent` tuple so ReBAC can follow it. Optional direct relations (e.g. `assignee`) add access on top.

```go
"task": {
	Relations: map[string][]string{
		"parent":   {"project"},   // structural, not shareable
		"assignee": {"user"},      // optional extra access
	},
	Permissions: map[string][]string{
		"view":   {"assignee", "parent->view"},
		"edit":   {"assignee", "parent->edit"},
		"delete": {"parent->edit"},
		"manage": {"parent->manage"},
	},
},
```

Model: `ProjectID uint gorm:"not null;index" json:"projectId"`, input `ProjectID uint binding:"required"`.

Service changes compared with the `project` example:

```go
func (s *Service) Create(ctx context.Context, p authz.Principal, in CreateInput) (*Task, error) {
	parent := authz.Obj("project", in.ProjectID)
	// Creating a child = editing the parent. No RBAC action needed.
	if err := s.az.Authorize(ctx, p, "edit", parent); err != nil {
		return nil, err
	}
	t := &Task{ProjectID: in.ProjectID, Title: strings.TrimSpace(in.Title), CreatedByID: p.UserID}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.WithTx(tx).Create(ctx, t); err != nil {
			return err
		}
		return s.az.WithTx(tx).Grant(ctx, authz.NewTuple(t.Object(), "parent", authz.Subject{Type: "project", ID: strconv.FormatUint(uint64(in.ProjectID), 10)}))
	})
	return t, err
}

// Lists are usually scoped to one parent: GET /api/projects/:id/tasks or /api/tasks?projectId=
func (s *Service) ListByProject(ctx context.Context, p authz.Principal, projectID uint, page httpx.Page) (httpx.List[Task], error) {
	if err := s.az.Authorize(ctx, p, "view", authz.Obj("project", projectID)); err != nil {
		return httpx.List[Task]{}, err
	}
	// Parent view implies view on all its tasks, so no per-row lookup is needed here.
	items, total, err := s.repo.List(ctx, ListFilter{Page: page, ProjectID: projectID, All: true})
	...
}
```

- Deleting a parent must delete its children (rows + `DeleteObject` for each) in the same transaction, or
  refuse with 409 while children exist. Pick one and say which in the spec.
- Moving a child to another parent = `Revoke` old parent tuple + `Grant` new one + update the column, in one transaction,
  after checking `edit` on both parents.
- If a parent may be deleted, `DeleteObject(parent)` also removes the children's `parent` tuples (subject match), so
  orphaned children become invisible — another reason to delete children explicitly.

## C. Admin-managed catalog (RBAC only)

Reference data everyone reads and only some roles edit: categories, locations, product types, settings.
No ReBAC type, no tuples.

```go
// rbac.go
RoleMember: {"team:create", "category:read"},
RoleViewer: {"category:read"},
// admin has "*"

// service
func (s *Service) List(ctx context.Context, p authz.Principal, page httpx.Page) (...) {
	// any signed-in user may read
	items, total, err := s.repo.List(ctx, ListFilter{Page: page, All: true})
}
func (s *Service) Create(ctx context.Context, p authz.Principal, in CreateInput) (*Category, error) {
	if !p.Can("category:manage") { return nil, httpx.Forbidden("admins only") }
	...
}
```

Frontend: show create/edit/delete when `useCan("category:manage")`; detail responses may return
`permissions: p.Can("category:manage") ? ["view","edit","delete"] : ["view"]` so pages stay uniform.

## D. Variants

- **Everyone in the org may read, only some may edit**: use the catalog pattern for reads (`List` with `All: true`,
  no view check) and keep ReBAC relations for `edit`/`delete`/`manage`.
- **Org-wide superuser role**: prefer granting actions in `RoleActions`; only extend `IsAdmin()` if that role
  must bypass every object check.
- **Approval flows** (e.g. `approver` relation): add a relation and a permission `approve: {"approver", "manage"}`,
  then guard a custom endpoint `POST /api/<things>/:id/approve` with `Authorize(..., "approve", ...)`.
