package authz

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TypeUser is the built-in subject type for end users.
const TypeUser = "user"

// Object identifies a protected resource, e.g. project:42.
type Object struct {
	Type string
	ID   string
}

func (o Object) String() string { return o.Type + ":" + o.ID }

// Obj builds an Object from a numeric primary key.
func Obj(objectType string, id uint) Object {
	return Object{Type: objectType, ID: strconv.FormatUint(uint64(id), 10)}
}

// Subject is who receives a relation: a user (user:7) or a userset (team:3#member = all members of team 3).
type Subject struct {
	Type     string
	ID       string
	Relation string
}

func (s Subject) String() string {
	if s.Relation == "" {
		return s.Type + ":" + s.ID
	}
	return s.Type + ":" + s.ID + "#" + s.Relation
}

func UserSubject(id uint) Subject {
	return Subject{Type: TypeUser, ID: strconv.FormatUint(uint64(id), 10)}
}

// ParseSubject parses "user:7" or "team:3#member".
func ParseSubject(s string) (Subject, error) {
	rest, rel, _ := strings.Cut(s, "#")
	typ, id, ok := strings.Cut(rest, ":")
	if !ok || !validName(typ) || !validID(id) || (rel != "" && !validName(rel)) {
		return Subject{}, fmt.Errorf("invalid subject %q (want type:id or type:id#relation)", s)
	}
	return Subject{Type: typ, ID: id, Relation: rel}, nil
}

// Tuple is one stored relationship: <object>#<relation>@<subject>,
// e.g. project:42#owner@user:7 or project:42#viewer@team:3#member.
type Tuple struct {
	ID              uint      `gorm:"primaryKey" json:"-"`
	ObjectType      string    `gorm:"size:64;not null;uniqueIndex:ux_relation_tuple,priority:1;index:ix_relation_tuple_subject,priority:4" json:"objectType"`
	ObjectID        string    `gorm:"size:64;not null;uniqueIndex:ux_relation_tuple,priority:2" json:"objectId"`
	Relation        string    `gorm:"size:64;not null;uniqueIndex:ux_relation_tuple,priority:3;index:ix_relation_tuple_subject,priority:5" json:"relation"`
	SubjectType     string    `gorm:"size:64;not null;uniqueIndex:ux_relation_tuple,priority:4;index:ix_relation_tuple_subject,priority:1" json:"subjectType"`
	SubjectID       string    `gorm:"size:64;not null;uniqueIndex:ux_relation_tuple,priority:5;index:ix_relation_tuple_subject,priority:2" json:"subjectId"`
	SubjectRelation string    `gorm:"size:64;not null;default:'';uniqueIndex:ux_relation_tuple,priority:6;index:ix_relation_tuple_subject,priority:3" json:"subjectRelation,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (Tuple) TableName() string { return "relation_tuples" }

func NewTuple(obj Object, relation string, sub Subject) Tuple {
	return Tuple{
		ObjectType:      obj.Type,
		ObjectID:        obj.ID,
		Relation:        relation,
		SubjectType:     sub.Type,
		SubjectID:       sub.ID,
		SubjectRelation: sub.Relation,
	}
}

func (t Tuple) Object() Object { return Object{Type: t.ObjectType, ID: t.ObjectID} }

func (t Tuple) Subject() Subject {
	return Subject{Type: t.SubjectType, ID: t.SubjectID, Relation: t.SubjectRelation}
}

func (t Tuple) String() string {
	return t.Object().String() + "#" + t.Relation + "@" + t.Subject().String()
}

// UintIDs converts object IDs from LookupResources into primary keys for `WHERE id IN ?`.
func UintIDs(ids []string) []uint {
	out := make([]uint, 0, len(ids))
	for _, s := range ids {
		if n, err := strconv.ParseUint(s, 10, 32); err == nil {
			out = append(out, uint(n))
		}
	}
	return out
}

func validName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func validID(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}
