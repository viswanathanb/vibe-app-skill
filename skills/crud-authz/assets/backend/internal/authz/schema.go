package authz

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Schema maps object type -> definition. See app_schema.go for the format.
type Schema map[string]TypeDef

type TypeDef struct {
	Relations   map[string][]string
	Permissions map[string][]string
}

type subjectRef struct {
	Type     string
	Relation string // set for usersets such as team#member
}

// term is one alternative in a permission: `name` on this object, or `name` on objects linked via `via`.
type term struct {
	name string
	via  string
}

type compiledType struct {
	relations   map[string][]subjectRef
	permissions map[string][]term
	permNames   []string
	shareable   []string
}

func (t *compiledType) has(name string) bool {
	_, rel := t.relations[name]
	_, perm := t.permissions[name]
	return rel || perm
}

func compile(s Schema) (map[string]*compiledType, error) {
	types := map[string]*compiledType{TypeUser: {relations: map[string][]subjectRef{}, permissions: map[string][]term{}}}
	for name, def := range s {
		if !validName(name) || name == TypeUser {
			return nil, fmt.Errorf("authz schema: invalid type name %q", name)
		}
		ct := &compiledType{relations: map[string][]subjectRef{}, permissions: map[string][]term{}}
		for rel, allowed := range def.Relations {
			if !validName(rel) || len(allowed) == 0 {
				return nil, fmt.Errorf("authz schema: %s.%s: invalid relation", name, rel)
			}
			for _, a := range allowed {
				typ, r, _ := strings.Cut(a, "#")
				ct.relations[rel] = append(ct.relations[rel], subjectRef{Type: typ, Relation: r})
			}
		}
		for perm, terms := range def.Permissions {
			if !validName(perm) || len(terms) == 0 {
				return nil, fmt.Errorf("authz schema: %s.%s: invalid permission", name, perm)
			}
			if _, clash := ct.relations[perm]; clash {
				return nil, fmt.Errorf("authz schema: %s.%s is both a relation and a permission", name, perm)
			}
			for _, raw := range terms {
				via, target, isArrow := strings.Cut(raw, "->")
				if !isArrow {
					ct.permissions[perm] = append(ct.permissions[perm], term{name: raw})
					continue
				}
				ct.permissions[perm] = append(ct.permissions[perm], term{name: target, via: via})
			}
			ct.permNames = append(ct.permNames, perm)
		}
		sort.Strings(ct.permNames)
		types[name] = ct
	}

	for name, ct := range types {
		for rel, refs := range ct.relations {
			userAssignable := false
			for _, ref := range refs {
				target, ok := types[ref.Type]
				if !ok {
					return nil, fmt.Errorf("authz schema: %s.%s allows unknown type %q", name, rel, ref.Type)
				}
				if ref.Relation != "" && !target.has(ref.Relation) {
					return nil, fmt.Errorf("authz schema: %s.%s allows %s#%s but %s has no %q",
						name, rel, ref.Type, ref.Relation, ref.Type, ref.Relation)
				}
				if ref.Type == TypeUser || ref.Relation != "" {
					userAssignable = true
				}
			}
			if userAssignable {
				ct.shareable = append(ct.shareable, rel)
			}
		}
		sort.Strings(ct.shareable)
		if len(ct.shareable) > 0 {
			if _, ok := ct.permissions["manage"]; !ok {
				return nil, fmt.Errorf("authz schema: %s has user-assignable relations but no `manage` permission", name)
			}
		}

		for perm, terms := range ct.permissions {
			for _, t := range terms {
				if t.via == "" {
					if t.name == perm || !ct.has(t.name) {
						return nil, fmt.Errorf("authz schema: %s.%s references unknown %q", name, perm, t.name)
					}
					continue
				}
				refs, ok := ct.relations[t.via]
				if !ok {
					return nil, fmt.Errorf("authz schema: %s.%s: %q is not a relation", name, perm, t.via)
				}
				linked := 0
				for _, ref := range refs {
					if ref.Relation != "" || ref.Type == TypeUser {
						continue
					}
					linked++
					if !types[ref.Type].has(t.name) {
						return nil, fmt.Errorf("authz schema: %s.%s: %s has no %q", name, perm, ref.Type, t.name)
					}
				}
				if linked == 0 {
					return nil, fmt.Errorf("authz schema: %s.%s: relation %q links no object types", name, perm, t.via)
				}
			}
		}
	}
	return types, nil
}

func (t *compiledType) allows(rel string, s Subject) bool {
	return slices.ContainsFunc(t.relations[rel], func(r subjectRef) bool {
		return r.Type == s.Type && r.Relation == s.Relation
	})
}
