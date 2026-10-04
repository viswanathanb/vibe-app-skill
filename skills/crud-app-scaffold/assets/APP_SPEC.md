# {{APP_NAME}} — app spec

Fill this in (or ask the agent to fill it in with you) before generating resources.
The agent builds one resource at a time from this file with the `crud-resource` skill.

## Purpose

One or two sentences: who uses the app and what for.

## Roles (RBAC)

Global roles are fixed: `admin` (everything), `member`, `viewer`. List what each may CREATE.

| Role | May create |
| --- | --- |
| admin | everything, manage users |
| member | teams, <resource>, ... |
| viewer | nothing (read what is shared with them) |

## Resources

Copy this block per resource. Order them parents first.

### <ResourceName> (plural: <resources>)

- **Description**:
- **Access pattern**: `owned` (creator owns it, can share with users/teams) | `child of <Parent>` (inherits access from parent) | `admin-managed catalog` (admins edit, everyone signed in reads)
- **Fields**:

| Field | Type | Required | Rules / notes |
| --- | --- | --- | --- |
| name | string | yes | max 200 |
| description | text | no | max 2000 |
| status | enum(active, archived) | yes | default active |

- **List page**: columns, search fields, filters, default sort
- **Detail page**: what to show, actions
- **Relations to share** (owned pattern): owner, editor, viewer (default)
- **Extra actions** (beyond CRUD), if any:

## Non-functional

- Expected data size:
- Anything that must NOT be possible:
