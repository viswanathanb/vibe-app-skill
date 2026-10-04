package authz

import (
	"context"
	"slices"
	"testing"
)

var testSchema = Schema{
	"team": {
		Relations:   map[string][]string{"admin": {"user"}, "member": {"user", "team#member"}},
		Permissions: map[string][]string{"view": {"member", "admin"}, "manage": {"admin"}},
	},
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
	"task": {
		Relations:   map[string][]string{"parent": {"project"}, "assignee": {"user"}},
		Permissions: map[string][]string{"view": {"assignee", "parent->view"}, "edit": {"assignee", "parent->edit"}, "manage": {"parent->manage"}},
	},
}

func newTestEngine(t *testing.T, tuples ...Tuple) *Engine {
	t.Helper()
	e, err := NewEngine(testSchema, NewMemoryStore())
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Grant(context.Background(), tuples...); err != nil {
		t.Fatal(err)
	}
	return e
}

func member(id uint) Principal { return Principal{UserID: id, Role: RoleMember} }

func TestCheck(t *testing.T) {
	ctx := context.Background()
	p1, p2 := Obj("project", 1), Obj("project", 2)
	team, subTeam := Obj("team", 10), Obj("team", 11)
	e := newTestEngine(t,
		NewTuple(p1, "owner", UserSubject(1)),
		NewTuple(p1, "viewer", Subject{Type: "team", ID: "10", Relation: "member"}),
		NewTuple(team, "member", Subject{Type: "team", ID: "11", Relation: "member"}), // nested team
		NewTuple(subTeam, "member", UserSubject(3)),
		NewTuple(p2, "editor", UserSubject(2)),
		NewTuple(Obj("task", 5), "parent", Subject{Type: "project", ID: "2"}),
	)

	cases := []struct {
		name string
		p    Principal
		perm string
		obj  Object
		want bool
	}{
		{"owner can delete", member(1), "delete", p1, true},
		{"owner can view via edit->manage", member(1), "view", p1, true},
		{"stranger cannot view", member(2), "view", p1, false},
		{"nested team member can view", member(3), "view", p1, true},
		{"nested team member cannot edit", member(3), "edit", p1, false},
		{"editor inherits to child task", member(2), "edit", Obj("task", 5), true},
		{"owner of other project gets nothing on task", member(1), "view", Obj("task", 5), false},
		{"admin bypasses", Principal{UserID: 99, Role: RoleAdmin}, "delete", p2, true},
		{"anonymous denied", Principal{}, "view", p1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.Check(ctx, tc.p, tc.perm, tc.obj)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("Check = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLookupResources(t *testing.T) {
	ctx := context.Background()
	e := newTestEngine(t,
		NewTuple(Obj("project", 1), "owner", UserSubject(1)),
		NewTuple(Obj("project", 2), "viewer", Subject{Type: "team", ID: "10", Relation: "member"}),
		NewTuple(Obj("team", 10), "member", Subject{Type: "team", ID: "11", Relation: "member"}),
		NewTuple(Obj("team", 11), "member", UserSubject(1)),
		NewTuple(Obj("project", 3), "editor", UserSubject(2)),
		NewTuple(Obj("task", 7), "parent", Subject{Type: "project", ID: "3"}),
	)

	ids, all, err := e.LookupResources(ctx, member(1), "view", "project")
	if err != nil || all {
		t.Fatalf("err=%v all=%v", err, all)
	}
	if want := []string{"1", "2"}; !slices.Equal(ids, want) {
		t.Fatalf("projects = %v, want %v", ids, want)
	}

	ids, _, err = e.LookupResources(ctx, member(2), "edit", "task")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"7"}; !slices.Equal(ids, want) {
		t.Fatalf("tasks = %v, want %v", ids, want)
	}

	if _, all, _ := e.LookupResources(ctx, Principal{UserID: 5, Role: RoleAdmin}, "view", "project"); !all {
		t.Fatal("admin should see all")
	}
}

func TestGrantValidation(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()
	bad := []Tuple{
		NewTuple(Obj("project", 1), "owner", Subject{Type: "team", ID: "1", Relation: "member"}),
		NewTuple(Obj("project", 1), "view", UserSubject(1)), // permission, not relation
		NewTuple(Obj("nope", 1), "owner", UserSubject(1)),
	}
	for _, tp := range bad {
		if err := e.Grant(ctx, tp); err == nil {
			t.Errorf("Grant(%s) succeeded, want error", tp)
		}
	}
}

func TestSchemaValidation(t *testing.T) {
	bad := []Schema{
		{"doc": {Relations: map[string][]string{"owner": {"user"}}, Permissions: map[string][]string{"view": {"owner"}}}}, // no manage
		{"doc": {Relations: map[string][]string{"owner": {"ghost"}}}},
		{"doc": {Relations: map[string][]string{"owner": {"user"}}, Permissions: map[string][]string{"manage": {"nope"}}}},
		{"doc": {Relations: map[string][]string{"owner": {"user"}}, Permissions: map[string][]string{"manage": {"owner->view"}}}},
	}
	for i, s := range bad {
		if _, err := NewEngine(s, NewMemoryStore()); err == nil {
			t.Errorf("schema %d: expected error", i)
		}
	}
	if _, err := NewEngine(AppSchema, NewMemoryStore()); err != nil {
		t.Fatalf("AppSchema invalid: %v", err)
	}
}
