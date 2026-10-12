// Command migrate applies or inspects customer-service's schema migrations, using
// the same DB_* environment as the server. The server applies them on
// startup unless DB_AUTO_MIGRATE=false; this is for running them on their
// own (a k8s Job, CI) or rolling one back.
//
//	go run ./cmd/migrate [up|down|status]   (default: up)
package main

import (
	"context"
	"customer-service/config"
	database_provider "customer-service/internal/adapters/database/provider"
	"customer-service/migrations"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	_ = godotenv.Load()
	db, err := database_provider.NewPostgresClient(*config.LoadConfig())
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	provider, err := migrations.NewProvider(sqlDB)
	if err != nil {
		return err
	}
	ctx := context.Background()

	switch command {
	case "up":
		results, err := provider.Up(ctx)
		for _, r := range results {
			fmt.Printf("applied %s (%s)\n", r.Source.Path, r.Duration)
		}
		if err == nil && len(results) == 0 {
			fmt.Println("no pending migrations")
		}
		return err
	case "down":
		r, err := provider.Down(ctx)
		if r != nil {
			fmt.Printf("rolled back %s\n", r.Source.Path)
		}
		return err
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			fmt.Printf("%-8s %s\n", s.State, s.Source.Path)
		}
		return nil
	}
	return fmt.Errorf("unknown command %q (want up, down or status)", command)
}
