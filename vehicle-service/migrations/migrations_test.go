package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var wantTables = []string{"vehicle_model", "vehicle", "customer_vehicle", "vehicle_material", "vehicle_service_history", "outbox_message"}

// newScratchDB creates an empty database on the server TEST_POSTGRES_URL
// points at (any database on it, e.g. postgres://postgres:postgres@localhost:5432/postgres)
// and drops it after the test. Skips if the variable isn't set.
func newScratchDB(t *testing.T) *sql.DB {
	t.Helper()
	admin := os.Getenv("TEST_POSTGRES_URL")
	if admin == "" {
		t.Skip("TEST_POSTGRES_URL not set")
	}
	adminDB, err := sql.Open("pgx", admin)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("migrations_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(admin)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		_ = adminDB.Close()
	})
	return db
}

func assertTables(t *testing.T, db *sql.DB, wantExist bool) {
	t.Helper()
	for _, table := range wantTables {
		var exists bool
		if err := db.QueryRow("SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != wantExist {
			t.Errorf("table %s exists = %v, want %v", table, exists, wantExist)
		}
	}
}

func TestUp_FreshDatabase(t *testing.T) {
	db := newScratchDB(t)
	ctx := context.Background()

	results, err := Up(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected migrations to be applied")
	}
	assertTables(t, db, true)

	again, err := Up(ctx, db)
	if err != nil || len(again) != 0 {
		t.Errorf("second Up applied %d migrations (err %v), want none", len(again), err)
	}
}

// TestUp_AdoptsExistingSchema simulates a volume built by the old init
// script: the baseline's statements without IF NOT EXISTS, run outside
// goose. The baseline must then apply cleanly on top.
func TestUp_AdoptsExistingSchema(t *testing.T) {
	db := newScratchDB(t)
	baseline, err := fs.ReadFile(FS, "00001_baseline.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.Split(string(baseline), "-- +goose Down")[0]
	legacy := regexp.MustCompile(`(?i) IF NOT EXISTS`).ReplaceAllString(up, "")
	if _, err := db.Exec(legacy); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}

	if _, err := Up(context.Background(), db); err != nil {
		t.Fatalf("Up on an existing schema: %v", err)
	}
	assertTables(t, db, true)
}

func TestDownThenUp(t *testing.T) {
	db := newScratchDB(t)
	ctx := context.Background()
	p, err := NewProvider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatal(err)
	}
	assertTables(t, db, false)
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	assertTables(t, db, true)
}
