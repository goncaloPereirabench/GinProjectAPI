package database

import (
	"context"
	"database/sql"
	"time"

	"ginprojectapi/internal/config"

	_ "github.com/microsoft/go-mssqldb"
)

func OpenSQLServer(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
