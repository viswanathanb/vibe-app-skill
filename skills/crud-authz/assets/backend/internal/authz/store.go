package authz

import (
	"context"
	"sort"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store persists relationship tuples.
type Store interface {
	// Subjects returns who holds relation on obj.
	Subjects(ctx context.Context, obj Object, relation string) ([]Subject, error)
	// Objects returns IDs of objectType where any of subjects holds relation.
	Objects(ctx context.Context, objectType, relation string, subjects []Subject) ([]string, error)
	List(ctx context.Context, obj Object) ([]Tuple, error)
	Write(ctx context.Context, tuples []Tuple) error
	Delete(ctx context.Context, tuples []Tuple) error
	DeleteObject(ctx context.Context, obj Object) error
}

// GormStore stores tuples in the relation_tuples table.
type GormStore struct{ db *gorm.DB }

func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

func (s *GormStore) WithTx(tx *gorm.DB) Store { return &GormStore{db: tx} }

func (s *GormStore) Subjects(ctx context.Context, obj Object, relation string) ([]Subject, error) {
	var rows []Tuple
	err := s.db.WithContext(ctx).
		Select("subject_type", "subject_id", "subject_relation").
		Where("object_type = ? AND object_id = ? AND relation = ?", obj.Type, obj.ID, relation).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Subject, len(rows))
	for i, r := range rows {
		out[i] = r.Subject()
	}
	return out, nil
}

func (s *GormStore) Objects(ctx context.Context, objectType, relation string, subjects []Subject) ([]string, error) {
	const chunk = 500
	var out []string
	for start := 0; start < len(subjects); start += chunk {
		part := subjects[start:min(start+chunk, len(subjects))]
		keys := make([][]any, len(part))
		for i, sub := range part {
			keys[i] = []any{sub.Type, sub.ID, sub.Relation}
		}
		var ids []string
		err := s.db.WithContext(ctx).Model(&Tuple{}).
			Where("object_type = ? AND relation = ?", objectType, relation).
			Where("(subject_type, subject_id, subject_relation) IN ?", keys).
			Distinct("object_id").
			Pluck("object_id", &ids).Error
		if err != nil {
			return nil, err
		}
		out = append(out, ids...)
	}
	return out, nil
}

func (s *GormStore) List(ctx context.Context, obj Object) ([]Tuple, error) {
	var rows []Tuple
	err := s.db.WithContext(ctx).
		Where("object_type = ? AND object_id = ?", obj.Type, obj.ID).
		Order("relation, subject_type, subject_id").
		Find(&rows).Error
	return rows, err
}

func (s *GormStore) Write(ctx context.Context, tuples []Tuple) error {
	if len(tuples) == 0 {
		return nil
	}
	rows := make([]Tuple, len(tuples))
	copy(rows, tuples)
	for i := range rows {
		rows[i].ID = 0
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (s *GormStore) Delete(ctx context.Context, tuples []Tuple) error {
	for _, t := range tuples {
		err := s.db.WithContext(ctx).
			Where("object_type = ? AND object_id = ? AND relation = ? AND subject_type = ? AND subject_id = ? AND subject_relation = ?",
				t.ObjectType, t.ObjectID, t.Relation, t.SubjectType, t.SubjectID, t.SubjectRelation).
			Delete(&Tuple{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *GormStore) DeleteObject(ctx context.Context, obj Object) error {
	return s.db.WithContext(ctx).
		Where("(object_type = ? AND object_id = ?) OR (subject_type = ? AND subject_id = ?)", obj.Type, obj.ID, obj.Type, obj.ID).
		Delete(&Tuple{}).Error
}

// MemoryStore is an in-memory Store for unit tests.
type MemoryStore struct {
	mu     sync.RWMutex
	tuples map[string]Tuple
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{tuples: map[string]Tuple{}} }

func (m *MemoryStore) Subjects(_ context.Context, obj Object, relation string) ([]Subject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Subject
	for _, t := range m.tuples {
		if t.ObjectType == obj.Type && t.ObjectID == obj.ID && t.Relation == relation {
			out = append(out, t.Subject())
		}
	}
	return out, nil
}

func (m *MemoryStore) Objects(_ context.Context, objectType, relation string, subjects []Subject) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	want := make(map[Subject]bool, len(subjects))
	for _, s := range subjects {
		want[s] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, t := range m.tuples {
		if t.ObjectType == objectType && t.Relation == relation && want[t.Subject()] && !seen[t.ObjectID] {
			seen[t.ObjectID] = true
			out = append(out, t.ObjectID)
		}
	}
	return out, nil
}

func (m *MemoryStore) List(_ context.Context, obj Object) ([]Tuple, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Tuple
	for _, t := range m.tuples {
		if t.ObjectType == obj.Type && t.ObjectID == obj.ID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

func (m *MemoryStore) Write(_ context.Context, tuples []Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range tuples {
		t.ID = 0
		m.tuples[t.String()] = t
	}
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, tuples []Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range tuples {
		delete(m.tuples, t.String())
	}
	return nil
}

func (m *MemoryStore) DeleteObject(_ context.Context, obj Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, t := range m.tuples {
		if (t.ObjectType == obj.Type && t.ObjectID == obj.ID) || (t.SubjectType == obj.Type && t.SubjectID == obj.ID) {
			delete(m.tuples, k)
		}
	}
	return nil
}
