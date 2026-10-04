package authz

// AppSchema is the ReBAC model for this app. EDIT THIS FILE when you add a protected resource type.
//
// Relations: relation name -> allowed subject types. Each entry is one of
//   - "user"         a single user              (project:1#owner@user:7)
//   - "team#member"  a userset: all team members (project:1#viewer@team:3#member)
//   - "project"      another object, for parent links (task:9#parent@project:1)
//
// Permissions: permission name -> terms; access is granted if ANY term matches:
//   - "owner"          a relation or another permission on the same object
//   - "parent->view"   follow relation `parent` to the linked object(s) and check `view` there
//
// Conventions used by the generic sharing API and the UI:
//   - every type with user-assignable relations defines `manage` (who may share / change access)
//   - CRUD services check `view`, `edit`, and `delete`
//
// NewEngine validates this schema at startup and refuses to start on mistakes.
var AppSchema = Schema{
	"team": {
		Relations: map[string][]string{
			"admin":  {"user"},
			"member": {"user"},
		},
		Permissions: map[string][]string{
			"view":   {"member", "admin"},
			"edit":   {"admin"},
			"delete": {"admin"},
			"manage": {"admin"},
		},
	},
	// Pattern A: owned + shareable top-level resource (users and whole teams can be granted access).
	// "project": {
	// 	Relations: map[string][]string{
	// 		"owner":  {"user"},
	// 		"editor": {"user", "team#member"},
	// 		"viewer": {"user", "team#member"},
	// 	},
	// 	Permissions: map[string][]string{
	// 		"view":   {"viewer", "edit"},
	// 		"edit":   {"editor", "manage"},
	// 		"delete": {"owner"},
	// 		"manage": {"owner"},
	// 	},
	// },
	// Pattern B: child resource that inherits access from its parent.
	// "task": {
	// 	Relations: map[string][]string{
	// 		"parent":   {"project"},
	// 		"assignee": {"user"},
	// 	},
	// 	Permissions: map[string][]string{
	// 		"view":   {"assignee", "parent->view"},
	// 		"edit":   {"assignee", "parent->edit"},
	// 		"delete": {"parent->edit"},
	// 		"manage": {"parent->manage"},
	// 	},
	// },
	// crud:schema (add new object types above this line)
}
