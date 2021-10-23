package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"os"
	"time"
)

// ConnectionConfig provides values for gRPC connection configuration
type ConnectionConfig struct {
	Host            string
	Port            int
	User            string
	Pass            string
	Database        string
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
}

// NewConnection provides new mysql connection
func NewConnection(ctx context.Context, cfg ConnectionConfig) (db *sql.DB) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", cfg.User, cfg.Pass, cfg.Host, cfg.Port, cfg.Database))
	if err != nil {
		logger.Critical(ctx, fmt.Sprintf("[MYSQL|Connection] %v", err))
		os.Exit(1)
	}

	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db
}
