# Field types

| Spec type | Go field + GORM tag | `binding` (CreateInput) | TS type | zod | Form control (shadcn) | List cell |
| --- | --- | --- | --- | --- | --- | --- |
| string (short) | `Name string gorm:"size:200;not null"` | `required,max=200` | `string` | `z.string().trim().min(1).max(200)` | `<Input>` | text / link |
| text (long) | `Notes string gorm:"size:10000;not null;default:''"` | `max=10000` | `string` | `z.string().max(10000)` | `<Textarea>` | truncated text |
| email | `Email string gorm:"size:320"` | `omitempty,email,max=320` | `string` | `z.email().or(z.literal(""))` | `<Input type="email">` | text |
| url | `URL string gorm:"size:2000"` | `omitempty,url,max=2000` | `string` | `z.url().or(z.literal(""))` | `<Input type="url">` | link |
| integer | `Replicas int gorm:"not null;default:0"` | `gte=0,lte=1000` | `number` | `z.coerce.number().int().min(0).max(1000)` | `<Input type="number">` | number |
| decimal / money | `Price int64` (store cents) | `gte=0` | `number` | `z.coerce.number().min(0)` → ×100 on submit | `<Input type="number" step="0.01">` | formatted |
| boolean | `Enabled bool gorm:"not null;default:false"` | — | `boolean` | `z.boolean()` | `<Switch>` / `<Checkbox>` via `Controller` | `<Badge>` |
| enum | `Status string gorm:"size:20;not null;default:active;index"` | `required,oneof=active archived` | `"active" \| "archived"` | `z.enum([...])` | `<Select>` via `Controller` | `<Badge>` |
| date | `DueOn *time.Time gorm:"type:date"` | — | `string \| null` | `z.string().optional()` (yyyy-mm-dd) | `<Input type="date">` | `toLocaleDateString` |
| datetime | `StartsAt *time.Time` | — | `string \| null` | `z.string().optional()` | `<Input type="datetime-local">` | `formatDate` |
| reference (FK) | `TeamID *uint gorm:"index"` | `omitempty,gt=0` | `number \| null` | `z.number().nullable()` | `<Select>` fed by the other feature's list hook | name via lookup |
| tags / list | `Tags datatypes.JSONSlice[string]` (gorm.io/datatypes) or a child table | `max=20,dive,max=50` | `string[]` | `z.array(z.string().max(50)).max(20)` | comma-separated `<Input>` | badges |
| JSON blob | `Config datatypes.JSON` | — | `unknown` | `z.string()` (validate JSON) | `<Textarea>` (monospace) | — |

Notes:

- Optional fields in `UpdateInput` are pointers (`*string`, `*int`, `*bool`, `*time.Time`). A nil pointer means
  "unchanged"; if a nullable field must be clearable, add an explicit `Clear<Field> bool` to the input.
- Numbers from `<Input type="number">` arrive as strings; use `z.coerce.number()` or `register("x", { valueAsNumber: true })`.
- Controlled shadcn inputs (`Select`, `Switch`, `Checkbox`) need `Controller` from react-hook-form
  (see `crud-auth` `CreateUserDialog.tsx` for a `Select` example). Add missing components with
  `cd frontend && bunx --bun shadcn@latest add switch checkbox`.
- For references, validate in the service that the referenced object exists **and** the caller may `view` it.
- Unique per scope (e.g. name unique per owner): composite `uniqueIndex:ux_<table>_<cols>` + catch
  `gorm.ErrDuplicatedKey` → `httpx.Conflict`.
- `gorm.io/datatypes` is an extra dependency; only add it when a field really needs JSON.
