package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/5afar/go_task_manager/internal/cache"
	"github.com/5afar/go_task_manager/internal/service"
	"github.com/5afar/go_task_manager/internal/store"
	pgstore "github.com/5afar/go_task_manager/internal/store/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func TestIntegration_PostgresRedis(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("could not connect to docker: %v", err)
	}

	// Postgres
	pgUser := "testuser"
	pgPass := "testpass"
	pgDB := "testdb"
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
		t.Fatalf("could not start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pool.Purge(pgRes) })

	var db *sql.DB
	dsn := ""
	// wait for Postgres
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
		t.Fatalf("could not connect to postgres: %v", err)
	}
	defer db.Close()

	// apply migrations
	mig, err := os.ReadFile("../../migrations/001_create_tasks.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(mig)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	// Redis
	redisPass := "redispass"
	redisRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "7-alpine",
		Cmd:        []string{"redis-server", "--requirepass", redisPass},
	})
	if err != nil {
		t.Fatalf("could not start redis container: %v", err)
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
		t.Fatalf("could not connect to redis: %v", err)
	}
	defer rcache.Close()

	// create repo and service
	repo, err := pgstore.New(dsn)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	defer repo.Close()
	svc := service.New(repo)

	// perform basic flow
	ctx := context.Background()
	task := &store.Task{Title: "integ", Description: "integration test"}
	if err := svc.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	got, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Title != task.Title {
		t.Fatalf("unexpected title: %s", got.Title)
	}

	// test redis cache
	if err := rcache.Set(ctx, "it:test", "ok", 5*time.Second); err != nil {
		t.Fatalf("redis set: %v", err)
	}
	v, err := rcache.Get(ctx, "it:test")
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	if v != "ok" {
		t.Fatalf("redis value mismatch: %s", v)
	}
}
