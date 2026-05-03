package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/5afar/go_task_manager/internal/service"
	"github.com/5afar/go_task_manager/internal/store"
)

// simple in-memory repo implementing store.TaskRepository for validation tests
type memRepo struct{}

func (m *memRepo) Create(_ context.Context, t *store.Task) error            { t.ID = "1"; return nil }
func (m *memRepo) GetByID(_ context.Context, _ string) (*store.Task, error) { return nil, nil }
func (m *memRepo) Update(_ context.Context, _ *store.Task) error            { return nil }
func (m *memRepo) Delete(_ context.Context, _ string) error                 { return nil }
func (m *memRepo) List(_ context.Context, _ int, _ int) ([]*store.Task, error) {
	return []*store.Task{}, nil
}

func TestCreateValidation(t *testing.T) {
	svc := service.New(&memRepo{})
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// create with empty title should return 400 and JSON error
	reqBody := map[string]string{"title": ""}
	b, _ := json.Marshal(reqBody)
	resp, err := http.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", resp.StatusCode)
	}
}
