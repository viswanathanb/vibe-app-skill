---
name: crud-authz
description: 'Authorization for crud-* apps: RBAC global roles (admin/member/viewer) mapped to app-wide actions, plus ReBAC (Zanzibar-style relation tuples stored in Postgres) for per-object access - owners, editors, viewers, sharing with users or teams, and inheritance from parent objects. Includes the schema DSL, Check/LookupResources/Grant/Revoke engine, generic sharing API and UI, and teams. Use when adding permissions, roles, sharing, ownership, "who can see/edit what", teams/groups, or securing list/detail endpoints.'
---

# Authorization: RBAC + ReBAC

Two small, complementary layers, all in Go + Postgres (no external service):

| Question | Mechanism | Where |
| --- | --- | --- |
| May this user do X **at all**? (create projects, manage users) | **RBAC**: `users.role` → actions | `internal/authz/rbac.go` |
| May this user do X **to this object**? (edit project 42) | **ReBAC**: relation tuples + schema | `internal/authz/app_schema.go`, table `relation_tuples` |

Admins (`role=admin`) pass every check. Unauthenticated principals fail every check.

## Concepts

- **Tuple**: `object#relation@subject`, e.g. `project:42#owner@user:7`, `project:42#viewer@team:3#member`
  (every member of team 3), `task:9#parent@project:42` (structural link).
- **Schema** (`AppSchema`): per object type, which subjects each **relation** accepts, and which relations
  grant each **permission**. Terms: `"owner"` (relation/permission on the same object) or `"parent->view"`
  (follow `parent` and check `view` there).
- Conventions: every type defines `view`, `edit`, `delete`; types with user-assignable relations must define
  `manage` (who can share). `NewEngine` validates the schema at startup and refuses to boot on mistakes.

```go
"project": {
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

## Engine API (`*authz.Engine`, built once in `server.New`)

| Call | Use for |
| --- | --- |
| `az.Authorize(ctx, p, "edit", authz.Obj("project", id))` | guard get/update/delete; returns 403 `httpx.Error` |
| `az.Check(...)` | same, returns bool |
| `az.LookupResources(ctx, p, "view", "project") (ids, all, err)` | filter lists: `all` → no filter (admin), else `WHERE id IN ids` |
| `az.Permissions(ctx, p, obj)` | return `permissions: [...]` in detail responses for the UI |
| `az.WithTx(tx).Grant(ctx, authz.NewTuple(obj, "owner", p.Subject()))` | on create, inside the same transaction |
| `az.WithTx(tx).DeleteObject(ctx, obj)` | on delete: removes tuples on the object **and** usersets pointing at it |
| `p.Can("project:create")` | RBAC check in services |
| `authz.RequireAction("users:manage")` | RBAC route middleware |

Sharing API (generic, every type): `GET /api/authz/:type/:id/permissions`, and with `manage`:
`GET|POST|DELETE /api/authz/:type/:id/tuples` — body `{relation, email}` or `{relation, subject:"team:3#member"}`;
revoke via `?relation=&subject=`. Only user-assignable relations can be shared; the last holder of a relation that
grants `manage` (e.g. the only owner) cannot be removed.

UI: `features/sharing/ShareDialog.tsx` (button + dialog) and `AccessPanel.tsx` (inline list). Show the Share button
only when `permissions` includes `manage`.

## Teams

`internal/team` + `features/teams` ship in the skeleton. Membership = tuples `team:<id>#member@user:<id>`, managed
with the same AccessPanel (team admins). Share anything with a team by granting a relation to `team:<id>#member`.
Deleting a team removes every share made to it. Nested teams work (`team#member` may contain `team#member`).

## Procedures

### Protect a new resource type

Pick a pattern from [references/patterns.md](./references/patterns.md) (owned / child / catalog), then:

1. Add the type to `AppSchema` above `// crud:schema`.
2. Add `"<type>:create"` to the roles that may create it in `RoleActions` (at `/* crud:actions */`), and
   `const ActionCreate = ObjectType + ":create"` in the resource package.
3. In the service: `p.Can(ActionCreate)` on create + grant owner/parent tuple in the create transaction;
   `Authorize` view/edit/delete; `LookupResources` for lists; `DeleteObject` in the delete transaction.
4. Add an engine test case if the rules are non-trivial (see `engine_test.go`, uses `NewMemoryStore`).

### Change who can do what

- Per object (e.g. "editors may delete"): edit the permission terms in `AppSchema` (`"delete": {"owner", "editor"}`).
- Per role (e.g. "viewers may create tickets"): add the action to `RoleActions[RoleViewer]`.
- New global role: add a constant to `Roles`, an entry in `RoleActions`, and update the `oneof=admin member viewer`
  binding tags in `user/model.go` plus the `Role` type and selects in the frontend.

### Remove teams (only if explicitly requested)

Delete `internal/team`, `features/teams`, their wiring/routes/nav/home card, the `"team"` schema entry and every
`team#member` allowed subject, and `"team:create"` from `RoleActions`.

## Performance notes

Checks run a few small indexed queries per request; list filtering does one query per relation hop. This is fine
for thousands of objects per user. If a list grows beyond ~10k visible IDs per user, add a denormalised
`owner_id`/`team_id` filter for the common case and keep ReBAC for detail checks.

## Verify

```bash
cd backend && go test ./internal/authz/...
```

Manual: user A creates an object; user B can't see it (list empty, detail 403); A shares with B as viewer →
B sees it but can't edit; A shares with a team containing B as editor → B can edit; delete the team → B loses access.
