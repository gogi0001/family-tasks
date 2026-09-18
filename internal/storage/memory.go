package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gogi0001/family-tasks/internal/models"
)

type Memory struct {
	mu    sync.RWMutex
	tasks map[string]*models.Task
}

func NewMemory() *Memory { return &Memory{tasks: make(map[string]*models.Task)} }

func (m *Memory) List(_ context.Context) ([]*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, clone(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) Get(_ context.Context, id string) (*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(t), nil
}

func (m *Memory) Create(_ context.Context, req models.CreateTaskRequest) (*models.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	t := &models.Task{
		ID: uuid.NewString(), Title: req.Title, Description: req.Description,
		Assignee: req.Assignee, CreatedBy: req.CreatedBy,
		Status: models.StatusTodo, CreatedAt: now, UpdatedAt: now,
	}
	m.tasks[t.ID] = t
	return clone(t), nil
}

func (m *Memory) Update(_ context.Context, id string, req models.UpdateTaskRequest) (*models.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
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
		t.Status = *req.Status
	}
	t.UpdatedAt = time.Now().UTC()
	return clone(t), nil
}

func (m *Memory) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

func clone(t *models.Task) *models.Task { c := *t; return &c }
