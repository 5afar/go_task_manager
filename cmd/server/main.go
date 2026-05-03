package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/5afar/go_task_manager/internal/api"
	"github.com/5afar/go_task_manager/internal/cache"
	"github.com/5afar/go_task_manager/internal/config"
	"github.com/5afar/go_task_manager/internal/logger"
	"github.com/5afar/go_task_manager/internal/service"
	"github.com/5afar/go_task_manager/internal/store/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func newCacheClient(cfg config.Config) (*cache.Client, error) {
	c := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// runMigrations applies SQL files from dir (alphabetical) and records them in schema_migrations
func runMigrations(dsn, dir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	files, err := listSQLFiles(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		name := filepath.Base(f)
		applied, err := isApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			logger.Infof("skip %s (already applied)", name)
			continue
		}
		logger.Infof("apply %s", name)
		if err := applyMigration(db, f, name); err != nil {
			return err
		}
		logger.Infof("applied %s", name)
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

func listSQLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".sql" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func isApplied(db *sql.DB, id string) (bool, error) {
	var exists bool
	row := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE id=$1)`, id)
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func applyMigration(db *sql.DB, path, id string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(string(b)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (id, applied_at) VALUES ($1, $2)`, id, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func healthHandler(c *cache.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := c.Ping(r.Context()); err != nil {
			http.Error(w, fmt.Sprintf("redis: %v", err), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	}
}

func cacheTestHandler(c *cache.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		key := "cache:test"
		val := fmt.Sprintf("value:%d", time.Now().UnixNano())
		if err := c.Set(ctx, key, val, 30*time.Second); err != nil {
			http.Error(w, fmt.Sprintf("set error: %v", err), http.StatusInternalServerError)
			return
		}
		got, err := c.Get(ctx, key)
		if err != nil {
			http.Error(w, fmt.Sprintf("get error: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cached=" + got + "\n"))
	}
}

func main() {
	cfg := config.Load()

	logger.Init("pet")

	// optional: run migrations at startup when AUTO_MIGRATE is set (true/1)
	if v := os.Getenv("AUTO_MIGRATE"); v == "1" || v == "true" {
		dsn := os.Getenv("POSTGRES_DSN")
		if dsn == "" {
			logger.Errorf("AUTO_MIGRATE set but POSTGRES_DSN is empty; skipping migrations")
		} else {
			logger.Infof("running migrations from migrations/ ...")
			if err := runMigrations(dsn, "migrations"); err != nil {
				logger.Fatalf("migrations failed: %v", err)
			}
			logger.Infof("migrations complete")
		}
	}

	c, err := newCacheClient(cfg)
	if err != nil {
		logger.Fatalf("failed to connect to redis: %v", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			logger.Errorf("redis close error: %v", err)
		}
	}()

	// initialize Postgres repo and service if DSN provided
	var svc *service.TaskService
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		repo, err := postgres.New(dsn)
		if err != nil {
			logger.Fatalf("failed to connect to postgres: %v", err)
		}
		defer func() { _ = repo.Close() }()
		svc = service.New(repo)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(c))
	mux.HandleFunc("/cache-test", cacheTestHandler(c))
	if svc != nil {
		api.RegisterRoutes(mux, svc)
	}

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: mux,
	}

	// start server
	go func() {
		logger.Infof("listening %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("server shutdown error: %v", err)
	}
	logger.Infof("shutting down")
}
