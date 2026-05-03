package store

import (
	"context"
	"time"
)

type Task struct {
	ID          string     `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status"`
	Priority    int        `json:"priority" db:"priority"`
	DueAt       *time.Time `json:"due_at,omitempty" db:"due_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	OwnerID     *string    `json:"owner_id,omitempty" db:"owner_id"`
}

// TaskRepository defines storage operations for tasks.
type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, t *Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*Task, error)
}
