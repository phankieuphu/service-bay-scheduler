// Package migrations holds scheduler-service's schema as numbered goose migrations
// (00001_baseline.sql, ...), embedded into the binary and applied on startup
// (see Up). Add a change as the next numbered file with "-- +goose Up" and
// "-- +goose Down" sections; never edit one that has shipped.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var embedded embed.FS

// FS is the migration files, for tests and tools.
var FS fs.FS = embedded

// NewProvider returns a goose provider for db. It holds a Postgres advisory
// lock while it migrates, so replicas starting together take turns instead
// of racing to apply the same migration.
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, db, FS, goose.WithSessionLocker(locker))
}

// Up applies every pending migration and returns what it applied.
func Up(ctx context.Context, db *sql.DB) ([]*goose.MigrationResult, error) {
	p, err := NewProvider(db)
	if err != nil {
		return nil, err
	}
	return p.Up(ctx)
}
