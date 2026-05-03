package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/5afar/go_task_manager/internal/store"
)

type fakeRepo struct {
	mu   sync.Mutex
	data map[string]*store.Task
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{data: make(map[string]*store.Task)}
}

func (f *fakeRepo) Create(ctx context.Context, t *store.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t.ID == "" {
		t.ID = "id-" + time.Now().Format("20060102150405.000000")
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	copy := *t
	f.data[t.ID] = &copy
	return nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id string) (*store.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.data[id]
	if t == nil {
		return nil, nil
	}
	copy := *t
	return &copy, nil
}

func (f *fakeRepo) Update(ctx context.Context, t *store.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data[t.ID] == nil {
		return sqlErrNoRows
	}
	t.UpdatedAt = time.Now().UTC()
	copy := *t
	f.data[t.ID] = &copy
	return nil
}

func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.data[id] == nil {
		return sqlErrNoRows
	}
	delete(f.data, id)
	return nil
}

func (f *fakeRepo) List(ctx context.Context, limit, offset int) ([]*store.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res := make([]*store.Task, 0, len(f.data))
	for _, v := range f.data {
		copy := *v
		res = append(res, &copy)
	}
	return res, nil
}

// sqlErrNoRows used to simulate sql.ErrNoRows behaviour in repo layer
var sqlErrNoRows = &fakeSQLError{}

type fakeSQLError struct{}

func (e *fakeSQLError) Error() string { return "no rows" }

func TestService_CreateGetUpdateDelete(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo)

	// Create
	t1 := &store.Task{Title: "Task 1", Description: "desc"}
	if err := svc.CreateTask(context.Background(), t1); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if t1.ID == "" {
		t.Fatalf("expected id to be set")
	}

	// Get
	got, err := svc.GetTask(context.Background(), t1.ID)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got.Title != t1.Title {
		t.Fatalf("got title %q want %q", got.Title, t1.Title)
	}

	// Update
	got.Title = "Task 1 updated"
	if err := svc.UpdateTask(context.Background(), got); err != nil {
		t.Fatalf("UpdateTask error: %v", err)
	}
	got2, _ := svc.GetTask(context.Background(), got.ID)
	if got2.Title != "Task 1 updated" {
		t.Fatalf("update failed: %q", got2.Title)
	}

	// Delete
	if err := svc.DeleteTask(context.Background(), got.ID); err != nil {
		t.Fatalf("DeleteTask error: %v", err)
	}
	if _, err := svc.GetTask(context.Background(), got.ID); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestService_Validation(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo)

	if err := svc.CreateTask(context.Background(), &store.Task{Title: ""}); err == nil {
		t.Fatalf("expected error for empty title")
	}
}
