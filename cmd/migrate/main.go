package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	// flags
	dir := flag.String("dir", "migrations", "migrations directory")
	dsn := flag.String("dsn", "", "Postgres DSN (overrides env POSTGRES_DSN)")
	flag.Parse()

	// load .env if present
	_ = godotenv.Load()

	finalDSN := *dsn
	if finalDSN == "" {
		finalDSN = os.Getenv("POSTGRES_DSN")
	}
	if finalDSN == "" {
		log.Fatal("Postgres DSN not provided; set POSTGRES_DSN or use -dsn flag")
	}

	db, err := sql.Open("pgx", finalDSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("ensure migrations table: %v", err)
	}

	files, err := listSQLFiles(*dir)
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}

	for _, f := range files {
		name := filepath.Base(f)
		applied, err := isApplied(db, name)
		if err != nil {
			log.Fatalf("checking migration %s: %v", name, err)
		}
		if applied {
			fmt.Printf("skip %s (already applied)\n", name)
			continue
		}
		fmt.Printf("apply %s\n", name)
		if err := applyMigration(db, f, name); err != nil {
			log.Fatalf("apply %s: %v", name, err)
		}
		fmt.Printf("applied %s\n", name)
	}

	fmt.Println("migrations complete")
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
	defer func() {
		// if still pending, rollback
		_ = tx.Rollback()
	}()
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
