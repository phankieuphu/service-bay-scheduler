package database_provider

import (
	"fmt"
	"identity-service/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresClient(cfg config.Config) (*gorm.DB, error) {
	dbCfg := cfg.Database
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.Username,
		dbCfg.Password,
		dbCfg.Database,
		dbCfg.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true, // cache prepare sql statement
	})
	if err != nil {
		return nil, err
	}

	// Configure underlying *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(dbCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(dbCfg.ConnMaxLifetime)

	// Pool stats as go_sql_* series (in-use/idle conns, wait_count_total,
	// wait_duration_seconds_total): a growing wait means requests queue for a
	// connection because DB_MAX_OPEN_CONNS is too low or Postgres is slow.
	// Register, not MustRegister, so a second client in tests doesn't panic.
	_ = prometheus.Register(collectors.NewDBStatsCollector(sqlDB, dbCfg.Database))

	return db, nil
}
