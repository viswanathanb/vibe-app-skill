---
name: crud-resource
description: 'Add one CRUD resource (entity) end-to-end to a crud-* app: GORM model, repository with search/sort/pagination, service with RBAC + ReBAC checks, Gin controller, wiring, authz schema entry, React list/detail pages, create/edit dialog, delete confirmation, sharing. Golden example is `project`; a script clones and wires it. Use when the user asks to add/create a new entity, table, model, resource, page, screen, or CRUD for X (e.g. "add clusters", "we need a Machine resource").'
argument-hint: 'Resource name (PascalCase) and its fields'
---

# Add a CRUD resource

Every resource is a copy of the golden example **`project`** ([backend](./assets/backend/internal/project/),
[frontend](./assets/frontend/src/features/projects/)) adapted to new fields. Never design a resource from scratch.

## Inputs (from `APP_SPEC.md` or ask)

- Name in PascalCase singular (+ plural if irregular): `Cluster`, `MachineConfig`/`MachineConfigs`, `Policy`/`Policies`.
- Fields with type, required, limits, enum values. See [field types](./references/field-types.md).
- Access pattern: **owned** (default), **child of <Parent>**, or **catalog**. See `crud-authz` → `references/patterns.md`.
- List columns, search fields, filters, default sort.

## Procedure

### 1. Generate (owned pattern, wired, compiles)

```bash
<skills-dir>/crud-resource/scripts/add-resource.sh <app-dir> <PascalName> [PascalPlural]
```

It creates `backend/internal/<pkg>/` and `frontend/src/features/<kebab-plural>/`, and inserts code at the markers:
`// crud:models`, `// crud:routes` + import (server.go), `/* crud:actions */` (member may create), `// crud:schema`
(owner/editor/viewer), `// crud:routes` (App.tsx), `// crud:nav` (AppLayout), `{/* crud:home */}` (HomePage).
It then runs gofmt, go build/vet, prettier, tsc and eslint.

Names it derives for `MachineConfig`: Go package `machineconfig`, authz type / action `machine_config` /
`machine_config:create`, URL + folder `machine-configs`, TS identifiers `MachineConfig`, `useMachineConfigs`.

If the script can't be used (e.g. not an app from these skills), do the same steps by hand following the example files.

### 2. Backend fields

`backend/internal/<pkg>/model.go`
- Replace `Name/Description/Status` with the spec's fields (keep `ID`, `CreatedByID`, `CreatedAt`, `UpdatedAt`).
- Mirror them in `CreateInput` (value types, `binding:"required,..."`) and `UpdateInput` (pointer types, `omitempty`).
- Enum: string column + constants + `binding:"oneof=a b c"`.

`repository.go`
- `sortColumns`: API key (camelCase) → column (snake_case) for sortable fields.
- Search: the `ILIKE` columns in `List`. Filters: add to `ListFilter` + `Where`.

`service.go`
- Copy new fields in `Create` (trim strings) and in `Update` (only when the pointer is non-nil).
- Business rules go here (e.g. uniqueness per owner → check + `httpx.Conflict`).
- Keep the authorization calls and both transactions exactly as generated.

`controller.go` – only touch it to read new query filters (`c.Query("status")`) or add custom actions.

### 3. Frontend fields

`frontend/src/features/<kebab-plural>/`
- `api.ts`: update the TS type (camelCase, dates are ISO strings), `<Name>Input`/`<Name>Update`.
- `<Name>FormDialog.tsx`: zod schema **matching the Go binding tags**, one input per field (see field types table).
- `<Plural>Page.tsx`: table columns, search placeholder, filters, default `sort`.
- `<Name>DetailPage.tsx`: fields shown; keep the `permissions`-based buttons (Share/Edit/Delete).
- Fix any labels the script couldn't humanise.

### 4. Access pattern adjustments

- **owned**: done.
- **child of Parent**: change schema to the child pattern, add `ParentID`, switch `Create` to "requires `edit` on parent"
  + grant `parent` tuple, scope lists by parent, and remove the `/* crud:actions */` entry. Usually show the child list
  inside the parent's detail page. Follow `crud-authz/references/patterns.md` §B.
- **catalog**: remove the schema entry, use `p.Can("<type>:manage")` for writes and `All: true` for reads. §C.
- Adjust which roles may create in `RoleActions` (`authz/rbac.go`). Viewers create nothing by default.

### 5. Verify

```bash
task lint && task test && task build
task dev   # then in the browser, with two accounts:
```

- A creates an item → appears in A's list; B's list is empty and B gets 403 on A's detail URL.
- A shares with B as viewer → B sees it, no Edit/Delete buttons, PATCH returns 403.
- Validation: submit an empty required field → inline error from zod; bypass with curl → 400 with `details`.
- Delete → gone for both; the tuples are gone too (`task db:psql`: `select * from relation_tuples where object_type='<type>';`).

Update `APP_SPEC.md` if decisions changed while implementing.

## Don'ts

- Don't skip `LookupResources` in `List` or `Authorize` in Get/Update/Delete — that leaks data.
- Don't build ORDER BY from raw query params; only via `sortColumns`.
- Don't add a second way to fetch data (no `fetch` in components, no global stores); use the feature's hooks.
- Don't rename the `crud:*` markers.
