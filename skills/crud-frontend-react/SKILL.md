---
name: crud-frontend-react
description: 'React frontend conventions for crud-* apps: Vite + React + TypeScript, Tailwind v4, shadcn/ui (Radix), TanStack Query data hooks, React Router routes, react-hook-form + zod forms, same-origin API client with CSRF header, auth guard, layout/nav, toasts, pagination, ESLint/Prettier/Vitest. Use when creating the frontend, adding pages/components/forms/tables, adding shadcn components, or fixing UI data fetching in an app built from these skills.'
---

# React Frontend (Vite + shadcn/ui + TanStack Query)

## Create the frontend

```bash
<skills-dir>/crud-frontend-react/scripts/create-frontend.sh <app-dir> "<App Name>"
```

Scaffolds Vite `react-ts`, swaps oxlint for ESLint, installs Tailwind v4, TanStack Query, React Router, react-hook-form,
zod, Prettier, Vitest; adds the `@/` alias; runs `shadcn init -t vite --base radix --preset nova`; adds the components
below; copies the skeleton UI (this skill + `crud-auth` + `crud-authz`); then checks format, lint, tests and build.

shadcn components installed: `button input label textarea card table dialog alert-dialog dropdown-menu badge select
sonner skeleton separator`. Add more with `bunx --bun shadcn@latest add <name>` (always `--base radix` components;
the project's `components.json` handles that).

## Layout

```
frontend/
  vite.config.ts           @ alias, Tailwind plugin, /api proxy -> :8080, vitest
  eslint.config.js         typescript-eslint + react-hooks + react-refresh (zero warnings)
  src/
    main.tsx               QueryClientProvider + <App/> + <Toaster/>
    App.tsx                route table (crud:routes marker)
    lib/api.ts             api.get/post/patch/put/delete, ApiError, Paginated<T>, toQuery
    lib/query-client.ts    defaults; 401 anywhere -> signed out
    lib/forms.ts           applyServerErrors (backend details -> form fields), formatDate
    components/            AppLayout (nav: crud:nav), RequireAuth, PageHeader, Pagination
    components/ui/         shadcn (generated - don't hand-edit unless restyling)
    pages/                 HomePage (crud:home), NotFoundPage
    features/<name>/       api.ts (types, query keys, hooks) + pages + dialogs
```

## Rules

- **Data**: only through feature hooks in `features/<name>/api.ts` using `api.*`. Never call `fetch` in components.
  Query keys: `xKeys.all / list(params) / detail(id)`. Mutations `invalidateQueries({ queryKey: xKeys.all })`.
  Lists use `placeholderData: keepPreviousData` to avoid flicker while paging.
- **Types** mirror the Go JSON (camelCase; timestamps are ISO strings; IDs are numbers).
- **Forms**: react-hook-form + `zodResolver`; zod rules mirror Go `binding` tags; on error call
  `applyServerErrors(err, setError)` so 400 `details` land on the right inputs. Use `Controller` for `Select`/`Switch`.
- **Permissions in the UI**:
  - RBAC: `const can = useCan(); can("project:create")` hides create buttons and nav items (`action` on `navItems`).
  - ReBAC: detail endpoints return `permissions`; show Edit/Delete/Share only when present.
  - UI checks are cosmetic. The backend enforces everything.
- **Feedback**: `toast.success/error` from `sonner`; destructive actions use `AlertDialog`.
- **Routing**: everything behind login goes under the `<RequireAuth/>` → `<AppLayout/>` branch in `App.tsx`.
- **Styling**: Tailwind utility classes + shadcn variants. Keep pages within `max-w-6xl` (AppLayout). No CSS files
  besides `index.css`.
- TypeScript is strict with `erasableSyntaxOnly`: no `enum`, no constructor parameter properties; use union types.

## Common tasks

- **New page**: create `features/<name>/<Page>.tsx`, add a route in `App.tsx` above `// crud:routes`, add a nav item
  above `// crud:nav` if it's top-level.
- **New field input**: see `crud-resource/references/field-types.md`.
- **Custom action button** (e.g. "Restart"): add `useRestartX` mutation in `api.ts` calling `POST /api/x/:id/restart`,
  guard it with a permission from `permissions`.
- **Code-splitting** (if the bundle warning bothers you): `lazy: () => import("@/features/x/XPage").then(m => ({ Component: m.XPage }))` in the route object.

## Dev server

`task dev` runs Vite on http://localhost:5173 and proxies `/api` to the Go API on :8080, so cookies are same-origin
in development and production. Never add CORS.

## Verify

```bash
cd frontend && bun run format:check && bun run lint && bun run typecheck && bun run test && bun run build
```
