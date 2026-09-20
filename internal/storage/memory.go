package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gogi0001/family-tasks/internal/models"
)

type Memory struct {
	mu       sync.RWMutex
	tasks    map[string]*models.Task
	users    map[string]*models.User // by id
	byName   map[string]string       // lower(name) -> user id
	families map[string]*models.Family
	byCode   map[string]string // invite_code -> family id
}

func NewMemory() *Memory {
	return &Memory{
		tasks:    make(map[string]*models.Task),
		users:    make(map[string]*models.User),
		byName:   make(map[string]string),
		families: make(map[string]*models.Family),
		byCode:   make(map[string]string),
	}
}

// --- tasks ---

func (m *Memory) List(_ context.Context, familyID string) ([]*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.Task, 0)
	for _, t := range m.tasks {
		if t.FamilyID == familyID {
			out = append(out, clone(t))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (m *Memory) Get(_ context.Context, familyID, id string) (*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	if !ok || t.FamilyID != familyID {
		return nil, ErrNotFound
	}
	return clone(t), nil
}

func (m *Memory) Create(_ context.Context, in models.TaskCreate) (*models.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	t := &models.Task{
		ID:          uuid.NewString(),
		FamilyID:    in.FamilyID,
		Title:       in.Title,
		Description: in.Description,
		Assignee:    in.Assignee,
		CreatedBy:   in.CreatedBy,
		Status:      models.StatusTodo,
		CreatedAt:   now,
		UpdatedAt:   now,
		DueAt:       in.DueAt,
	}
	m.tasks[t.ID] = t
	return clone(t), nil
}

func (m *Memory) Update(_ context.Context, familyID, id, actorID string, req models.UpdateTaskRequest) (*models.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok || t.FamilyID != familyID {
		return nil, ErrNotFound
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Assignee != nil {
		t.Assignee = *req.Assignee
	}
	if req.Status != nil {
		now := time.Now().UTC()
		t.Status = *req.Status
		t.StatusUpdatedBy = actorID
		t.StatusUpdatedAt = &now
	}
	if req.DueAt != nil {
		if *req.DueAt == "" {
			t.DueAt = nil
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.DueAt)
			if err != nil {
				return nil, fmt.Errorf("parse dueAt: %w", err)
			}
			u := parsed.UTC()
			t.DueAt = &u
		}
	}
	t.UpdatedAt = time.Now().UTC()
	return clone(t), nil
}

func (m *Memory) Delete(_ context.Context, familyID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok || t.FamilyID != familyID {
		return ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

// --- users ---

func (m *Memory) GetUser(_ context.Context, id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *u
	if c.Color == "" {
		c.Color = DefaultColorFor(c.Name)
	}
	return &c, nil
}

func (m *Memory) FindOrCreateByName(_ context.Context, name string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(name)
	if id, ok := m.byName[key]; ok {
		c := *m.users[id]
		if c.Color == "" {
			c.Color = DefaultColorFor(c.Name)
		}
		return &c, nil
	}
	u := &models.User{
		ID:        uuid.NewString(),
		Name:      name,
		Color:     DefaultColorFor(name),
		CreatedAt: time.Now().UTC(),
	}
	m.users[u.ID] = u
	m.byName[key] = u.ID
	c := *u
	return &c, nil
}

func (m *Memory) SetFamily(_ context.Context, userID, familyID string, role models.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	u.FamilyID = familyID
	u.Role = role
	return nil
}

func (m *Memory) SetColor(_ context.Context, userID, color string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	u.Color = color
	return nil
}

// --- families ---

func (m *Memory) CreateFamily(_ context.Context, name, ownerID string) (*models.Family, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := &models.Family{
		ID:         uuid.NewString(),
		Name:       name,
		InviteCode: generateInviteCode(),
		CreatedBy:  ownerID,
		CreatedAt:  time.Now().UTC(),
	}
	m.families[f.ID] = f
	m.byCode[f.InviteCode] = f.ID
	c := *f
	return &c, nil
}

func (m *Memory) GetFamilyByID(_ context.Context, id string) (*models.Family, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.families[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *f
	return &c, nil
}

func (m *Memory) GetFamilyByCode(_ context.Context, code string) (*models.Family, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byCode[strings.ToUpper(code)]
	if !ok {
		return nil, ErrNotFound
	}
	c := *m.families[id]
	return &c, nil
}

func (m *Memory) FamilyMembers(_ context.Context, familyID string) ([]models.FamilyMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]models.FamilyMember, 0)
	for _, u := range m.users {
		if u.FamilyID == familyID {
			color := u.Color
			if color == "" {
				color = DefaultColorFor(u.Name)
			}
			out = append(out, models.FamilyMember{
				ID:    u.ID,
				Name:  u.Name,
				Role:  u.Role,
				Color: color,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// --- helpers ---

func clone(t *models.Task) *models.Task { c := *t; return &c }

func (m *Memory) RegenerateInviteCode(_ context.Context, familyID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.families[familyID]
	if !ok {
		return "", ErrNotFound
	}
	delete(m.byCode, f.InviteCode)
	f.InviteCode = generateInviteCode()
	m.byCode[f.InviteCode] = familyID
	return f.InviteCode, nil
}

func (m *Memory) RemoveFromFamily(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return ErrNotFound
	}
	u.FamilyID = ""
	u.Role = ""
	return nil
}
