package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/5afar/go_task_manager/internal/store"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestRepo_CRUD_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := &Repo{db: db}
	ctx := context.Background()

	// CREATE
	// We allow any args and return success
	mock.ExpectExec("INSERT INTO tasks").WillReturnResult(sqlmock.NewResult(1, 1))

	t1 := &store.Task{Title: "t1", Description: "d1"}
	if err := repo.Create(ctx, t1); err != nil {
		t.Fatalf("create error: %v", err)
	}

	// GET
	cols := []string{"id", "title", "description", "status", "priority", "due_at", "created_at", "updated_at", "owner_id"}
	now := time.Now()
	mock.ExpectQuery("SELECT id, title, description, status, priority, due_at, created_at, updated_at, owner_id").WithArgs("id-1").WillReturnRows(sqlmock.NewRows(cols).AddRow("id-1", "t1", "d1", "todo", 0, nil, now, now, nil))
	got, err := repo.GetByID(ctx, "id-1")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got == nil || got.Title != "t1" {
		t.Fatalf("unexpected get result: %#v", got)
	}

	// UPDATE
	mock.ExpectExec("UPDATE tasks").WillReturnResult(sqlmock.NewResult(0, 1))
	got.Title = "updated"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update error: %v", err)
	}

	// DELETE
	mock.ExpectExec("DELETE FROM tasks").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// LIST
	mock.ExpectQuery("SELECT id, title, description, status, priority, due_at, created_at, updated_at, owner_id").WillReturnRows(sqlmock.NewRows(cols).AddRow("id-2", "t2", "d2", "todo", 0, nil, now, now, nil))
	list, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "id-2" {
		t.Fatalf("unexpected list result: %#v", list)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
