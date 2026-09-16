package database_provider

import (
	"fmt"
	"vehicle-service/config"

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

	return db, nil
}
