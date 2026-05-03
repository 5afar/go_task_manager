package service

import (
	"context"
	"errors"
	"strings"

	"github.com/5afar/go_task_manager/internal/store"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrInvalidTask = errors.New("invalid task")
)

type TaskService struct {
	repo store.TaskRepository
}

func New(repo store.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) CreateTask(ctx context.Context, t *store.Task) error {
	if strings.TrimSpace(t.Title) == "" {
		return ErrInvalidTask
	}
	if t.Status == "" {
		t.Status = "todo"
	}
	return s.repo.Create(ctx, t)
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*store.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, t *store.Task) error {
	if t.ID == "" {
		return ErrInvalidTask
	}
	if strings.TrimSpace(t.Title) == "" {
		return ErrInvalidTask
	}
	return s.repo.Update(ctx, t)
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context, limit, offset int) ([]*store.Task, error) {
	return s.repo.List(ctx, limit, offset)
}
