package authz

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"gorm.io/gorm"

	"example.com/app/internal/httpx"
)

const maxDepth = 32

var ErrMaxDepth = errors.New("authz: maximum evaluation depth exceeded")

// Engine evaluates the schema against stored tuples (a small Zanzibar-style checker).
type Engine struct {
	types map[string]*compiledType
	store Store
}

func NewEngine(schema Schema, store Store) (*Engine, error) {
	types, err := compile(schema)
	if err != nil {
		return nil, err
	}
	return &Engine{types: types, store: store}, nil
}

// WithTx returns an engine whose writes join the given GORM transaction.
func (e *Engine) WithTx(tx *gorm.DB) *Engine {
	if ts, ok := e.store.(interface{ WithTx(*gorm.DB) Store }); ok {
		return &Engine{types: e.types, store: ts.WithTx(tx)}
	}
	return e
}

// Check reports whether p has permission (or relation) `name` on obj. Admins always pass.
func (e *Engine) Check(ctx context.Context, p Principal, name string, obj Object) (bool, error) {
	if p.IsAdmin() {
		return true, nil
	}
	if p.UserID == 0 {
		return false, nil
	}
	if err := e.validateName(obj.Type, name); err != nil {
		return false, err
	}
	return e.check(ctx, p.Subject(), obj, name, map[string]bool{}, 0)
}

// Authorize is Check that returns a 403 error when access is denied.
func (e *Engine) Authorize(ctx context.Context, p Principal, name string, obj Object) error {
	ok, err := e.Check(ctx, p, name, obj)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.Forbidden(fmt.Sprintf("you do not have %s access to this %s", name, obj.Type))
	}
	return nil
}

// Permissions lists every permission p holds on obj (for the UI to show/hide actions).
func (e *Engine) Permissions(ctx context.Context, p Principal, obj Object) ([]string, error) {
	t, ok := e.types[obj.Type]
	if !ok {
		return nil, fmt.Errorf("authz: unknown type %q", obj.Type)
	}
	out := []string{}
	for _, perm := range t.permNames {
		ok, err := e.Check(ctx, p, perm, obj)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, perm)
		}
	}
	return out, nil
}

func (e *Engine) check(ctx context.Context, sub Subject, obj Object, name string, seen map[string]bool, depth int) (bool, error) {
	if depth > maxDepth {
		return false, ErrMaxDepth
	}
	key := obj.String() + "#" + name
	if seen[key] {
		return false, nil // already evaluated (false) or a cycle in progress
	}
	seen[key] = true

	t, ok := e.types[obj.Type]
	if !ok {
		return false, fmt.Errorf("authz: unknown type %q", obj.Type)
	}

	if terms, isPerm := t.permissions[name]; isPerm {
		for _, tm := range terms {
			if tm.via == "" {
				if ok, err := e.check(ctx, sub, obj, tm.name, seen, depth+1); err != nil || ok {
					return ok, err
				}
				continue
			}
			linked, err := e.store.Subjects(ctx, obj, tm.via)
			if err != nil {
				return false, err
			}
			for _, l := range linked {
				if l.Relation != "" || l.Type == TypeUser {
					continue
				}
				if ok, err := e.check(ctx, sub, Object{Type: l.Type, ID: l.ID}, tm.name, seen, depth+1); err != nil || ok {
					return ok, err
				}
			}
		}
		return false, nil
	}

	subjects, err := e.store.Subjects(ctx, obj, name)
	if err != nil {
		return false, err
	}
	for _, s := range subjects {
		if s.Relation == "" {
			if s.Type == sub.Type && s.ID == sub.ID {
				return true, nil
			}
			continue
		}
		if ok, err := e.check(ctx, sub, Object{Type: s.Type, ID: s.ID}, s.Relation, seen, depth+1); err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

// LookupResources returns the IDs of objectType that p has permission `name` on.
// all=true means "no filter" (admins). Use it to filter list queries: WHERE id IN ids.
func (e *Engine) LookupResources(ctx context.Context, p Principal, name, objectType string) (ids []string, all bool, err error) {
	if p.IsAdmin() {
		return nil, true, nil
	}
	if p.UserID == 0 {
		return []string{}, false, nil
	}
	if err := e.validateName(objectType, name); err != nil {
		return nil, false, err
	}
	l := &lookup{e: e, sub: p.Subject(), memo: map[string]idSet{}}
	var set idSet
	for {
		l.done, l.active, l.changed, l.cyclic = map[string]bool{}, map[string]bool{}, false, false
		if set, err = l.resolve(ctx, objectType, name, 0); err != nil {
			return nil, false, err
		}
		// Recursive usersets (e.g. nested teams) need another pass until results stop growing.
		if !l.cyclic || !l.changed {
			break
		}
	}
	ids = make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, false, nil
}

type idSet map[string]struct{}

type lookup struct {
	e       *Engine
	sub     Subject
	memo    map[string]idSet
	done    map[string]bool
	active  map[string]bool
	changed bool
	cyclic  bool
}

func (l *lookup) resolve(ctx context.Context, typ, name string, depth int) (idSet, error) {
	if depth > maxDepth {
		return nil, ErrMaxDepth
	}
	key := typ + "#" + name
	if l.done[key] {
		return l.memo[key], nil
	}
	if l.active[key] {
		l.cyclic = true
		return l.memo[key], nil
	}
	l.active[key] = true
	defer delete(l.active, key)

	t := l.e.types[typ]
	out := idSet{}
	for id := range l.memo[key] {
		out[id] = struct{}{}
	}

	if terms, isPerm := t.permissions[name]; isPerm {
		for _, tm := range terms {
			if tm.via == "" {
				ids, err := l.resolve(ctx, typ, tm.name, depth+1)
				if err != nil {
					return nil, err
				}
				union(out, ids)
				continue
			}
			for _, ref := range t.relations[tm.via] {
				if ref.Relation != "" || ref.Type == TypeUser {
					continue
				}
				parents, err := l.resolve(ctx, ref.Type, tm.name, depth+1)
				if err != nil {
					return nil, err
				}
				if err := l.collect(ctx, out, typ, tm.via, ref.Type, "", parents); err != nil {
					return nil, err
				}
			}
		}
	} else {
		for _, ref := range t.relations[name] {
			switch {
			case ref.Relation == "" && ref.Type == l.sub.Type:
				ids, err := l.e.store.Objects(ctx, typ, name, []Subject{l.sub})
				if err != nil {
					return nil, err
				}
				for _, id := range ids {
					out[id] = struct{}{}
				}
			case ref.Relation != "":
				groups, err := l.resolve(ctx, ref.Type, ref.Relation, depth+1)
				if err != nil {
					return nil, err
				}
				if err := l.collect(ctx, out, typ, name, ref.Type, ref.Relation, groups); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(out) > len(l.memo[key]) {
		l.changed = true
	}
	l.memo[key] = out
	l.done[key] = true
	return out, nil
}

// collect adds objects of typ that have `relation` to any subject in subjectIDs.
func (l *lookup) collect(ctx context.Context, out idSet, typ, relation, subjectType, subjectRel string, subjectIDs idSet) error {
	if len(subjectIDs) == 0 {
		return nil
	}
	subs := make([]Subject, 0, len(subjectIDs))
	for id := range subjectIDs {
		subs = append(subs, Subject{Type: subjectType, ID: id, Relation: subjectRel})
	}
	ids, err := l.e.store.Objects(ctx, typ, relation, subs)
	if err != nil {
		return err
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return nil
}

func union(dst, src idSet) {
	for id := range src {
		dst[id] = struct{}{}
	}
}

// Grant stores relationship tuples after validating them against the schema (idempotent).
func (e *Engine) Grant(ctx context.Context, tuples ...Tuple) error {
	for _, t := range tuples {
		if err := e.validateTuple(t); err != nil {
			return err
		}
	}
	return e.store.Write(ctx, tuples)
}

// Revoke deletes relationship tuples (missing tuples are ignored).
func (e *Engine) Revoke(ctx context.Context, tuples ...Tuple) error {
	return e.store.Delete(ctx, tuples)
}

// DeleteObject removes every tuple on obj and every tuple granting obj's usersets.
// Call it in the same transaction that deletes the resource.
func (e *Engine) DeleteObject(ctx context.Context, obj Object) error {
	return e.store.DeleteObject(ctx, obj)
}

// Tuples lists the tuples stored on obj.
func (e *Engine) Tuples(ctx context.Context, obj Object) ([]Tuple, error) {
	return e.store.List(ctx, obj)
}

// Subjects lists who holds `relation` directly on obj.
func (e *Engine) Subjects(ctx context.Context, obj Object, relation string) ([]Subject, error) {
	return e.store.Subjects(ctx, obj, relation)
}

// ShareableRelations lists relations that can be granted to users or usersets via the sharing API.
func (e *Engine) ShareableRelations(objectType string) []string {
	if t, ok := e.types[objectType]; ok {
		return t.shareable
	}
	return nil
}

// HasType reports whether objectType is declared in the schema.
func (e *Engine) HasType(objectType string) bool {
	_, ok := e.types[objectType]
	return ok && objectType != TypeUser
}

func (e *Engine) validateName(objectType, name string) error {
	t, ok := e.types[objectType]
	if !ok {
		return fmt.Errorf("authz: unknown type %q", objectType)
	}
	if !t.has(name) {
		return fmt.Errorf("authz: %s has no relation or permission %q", objectType, name)
	}
	return nil
}

func (e *Engine) validateTuple(tp Tuple) error {
	t, ok := e.types[tp.ObjectType]
	if !ok || tp.ObjectType == TypeUser {
		return httpx.BadRequest(fmt.Sprintf("unknown object type %q", tp.ObjectType))
	}
	if !validID(tp.ObjectID) || !validID(tp.SubjectID) {
		return httpx.BadRequest("invalid object or subject id")
	}
	if _, ok := t.relations[tp.Relation]; !ok {
		return httpx.BadRequest(fmt.Sprintf("%s has no relation %q", tp.ObjectType, tp.Relation))
	}
	if !t.allows(tp.Relation, tp.Subject()) {
		return httpx.BadRequest(fmt.Sprintf("%s#%s cannot be granted to %s", tp.ObjectType, tp.Relation, tp.Subject()))
	}
	return nil
}
