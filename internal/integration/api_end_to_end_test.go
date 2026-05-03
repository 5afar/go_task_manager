package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/5afar/go_task_manager/internal/api"
	"github.com/5afar/go_task_manager/internal/cache"
	"github.com/5afar/go_task_manager/internal/service"
	"github.com/5afar/go_task_manager/internal/store"
	pgstore "github.com/5afar/go_task_manager/internal/store/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func TestAPI_EndToEnd(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("docker pool: %v", err)
	}

	// Start Postgres
	pgUser, pgPass, pgDB := "testuser", "testpass", "testdb"
	pgRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_USER=" + pgUser,
			"POSTGRES_PASSWORD=" + pgPass,
			"POSTGRES_DB=" + pgDB,
		},
	}, func(hostConfig *docker.HostConfig) {
		hostConfig.AutoRemove = true
		hostConfig.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Purge(pgRes) })

	var db *sql.DB
	var dsn string
	if err := pool.Retry(func() error {
		hostPort := pgRes.GetPort("5432/tcp")
		dsn = fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s?sslmode=disable", pgUser, pgPass, hostPort, pgDB)
		var err error
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("connect pg: %v", err)
	}
	defer db.Close()

	// Apply migration
	mig, err := os.ReadFile("../../migrations/001_create_tasks.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(mig)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Start Redis
	redisPass := "redispass"
	redisRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "7-alpine",
		Cmd:        []string{"redis-server", "--requirepass", redisPass},
	})
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	t.Cleanup(func() { _ = pool.Purge(redisRes) })

	var rcache *cache.Client
	if err := pool.Retry(func() error {
		hostPort := redisRes.GetPort("6379/tcp")
		addr := fmt.Sprintf("localhost:%s", hostPort)
		c := cache.New(addr, redisPass, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := c.Ping(ctx); err != nil {
			return err
		}
		rcache = c
		return nil
	}); err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	defer rcache.Close()

	// Repo and service
	repo, err := pgstore.New(dsn)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	defer repo.Close()
	svc := service.New(repo)

	// HTTP server
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, svc)
	// add cache-test endpoint similar to cmd/server
	mux.HandleFunc("/cache-test", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key := "cache:test"
		val := "val"
		if err := rcache.Set(ctx, key, val, 30*time.Second); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		got, err := rcache.Get(ctx, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(got))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := ts.Client()

	// Create task
	t.Run("create-get-list-update-delete", func(t *testing.T) {
		newTask := &store.Task{Title: "it", Description: "api test"}
		b, _ := json.Marshal(newTask)
		resp, err := client.Post(ts.URL+"/tasks", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("post task: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201; got %d", resp.StatusCode)
		}
		var created store.Task
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatalf("decode created: %v", err)
		}
		resp.Body.Close()
		if created.ID == "" {
			t.Fatalf("empty id")
		}

		// get
		resp, err = client.Get(ts.URL + "/tasks/" + created.ID)
		if err != nil {
			t.Fatalf("get task: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200; got %d", resp.StatusCode)
		}
		var got store.Task
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode get: %v", err)
		}
		resp.Body.Close()
		if got.Title != newTask.Title {
			t.Fatalf("title mismatch")
		}

		// list
		resp, err = client.Get(ts.URL + "/tasks")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for list; got %d", resp.StatusCode)
		}
		var list []store.Task
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		resp.Body.Close()
		if len(list) == 0 {
			t.Fatalf("expected at least one task")
		}

		// update
		updated := &store.Task{Title: "updated", Description: "changed"}
		ub, _ := json.Marshal(updated)
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/tasks/"+created.ID, bytes.NewReader(ub))
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("update req: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 update; got %d", resp.StatusCode)
		}
		var up store.Task
		if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
			t.Fatalf("decode update: %v", err)
		}
		resp.Body.Close()
		if up.Title != "updated" {
			t.Fatalf("update failed")
		}

		// delete
		req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/tasks/"+created.ID, nil)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("delete req: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 delete; got %d", resp.StatusCode)
		}

		// get after delete -> 404
		resp, err = client.Get(ts.URL + "/tasks/" + created.ID)
		if err != nil {
			t.Fatalf("get after delete: %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 after delete; got %d", resp.StatusCode)
		}
	})

	// cache endpoint test
	t.Run("cache-test", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/cache-test")
		if err != nil {
			t.Fatalf("cache-test request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("cache-test status: %d", resp.StatusCode)
		}
		var body bytes.Buffer
		_, _ = body.ReadFrom(resp.Body)
		resp.Body.Close()
		if body.String() != "val" {
			t.Fatalf("cache value mismatch: %s", body.String())
		}
	})
}
