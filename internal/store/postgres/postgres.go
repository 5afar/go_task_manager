package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/5afar/go_task_manager/internal/store"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repo struct {
	db *sql.DB
}

func New(dsn string) (*Repo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Repo{db: db}, nil
}

func (r *Repo) Close() error {
	return r.db.Close()
}

func (r *Repo) Create(ctx context.Context, t *store.Task) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
        INSERT INTO tasks (id, title, description, status, priority, due_at, created_at, updated_at, owner_id)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
    `, t.ID, t.Title, t.Description, t.Status, t.Priority, t.DueAt, t.CreatedAt, t.UpdatedAt, t.OwnerID)
	return err
}

func (r *Repo) GetByID(ctx context.Context, id string) (*store.Task, error) {
	row := r.db.QueryRowContext(ctx, `
        SELECT id, title, description, status, priority, due_at, created_at, updated_at, owner_id
        FROM tasks WHERE id = $1
    `, id)
	var t store.Task
	var due sql.NullTime
	var owner sql.NullString
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &due, &t.CreatedAt, &t.UpdatedAt, &owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if due.Valid {
		tt := due.Time
		t.DueAt = &tt
	}
	if owner.Valid {
		s := owner.String
		t.OwnerID = &s
	}
	return &t, nil
}

func (r *Repo) Update(ctx context.Context, t *store.Task) error {
	t.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
        UPDATE tasks SET title=$1, description=$2, status=$3, priority=$4, due_at=$5, updated_at=$6, owner_id=$7 WHERE id=$8
    `, t.Title, t.Description, t.Status, t.Priority, t.DueAt, t.UpdatedAt, t.OwnerID, t.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repo) List(ctx context.Context, limit, offset int) ([]*store.Task, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, title, description, status, priority, due_at, created_at, updated_at, owner_id
        FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2
    `, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*store.Task
	for rows.Next() {
		var t store.Task
		var due sql.NullTime
		var owner sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &due, &t.CreatedAt, &t.UpdatedAt, &owner); err != nil {
			return nil, err
		}
		if due.Valid {
			tt := due.Time
			t.DueAt = &tt
		}
		if owner.Valid {
			s := owner.String
			t.OwnerID = &s
		}
		res = append(res, &t)
	}
	return res, rows.Err()
}
