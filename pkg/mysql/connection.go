package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/vardius/golog"
)

// ConnectionConfig provides values for gRPC connection configuration
type ConnectionConfig interface {
	GetMysqlHost() string
	GetMysqlPort() int
	GetMysqlUser() string
	GetMysqlPass() string
	GetMysqlDatabase() string
	GetMysqlConnMaxLifetime() time.Duration
	GetMysqlMaxIdleConns() int
	GetMysqlMaxOpenConns() int
}

// NewConnection provides new mysql connection
func NewConnection(ctx context.Context, cfg ConnectionConfig, logger golog.Logger) (db *sql.DB) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", cfg.GetMysqlUser(), cfg.GetMysqlPass(), cfg.GetMysqlHost(), cfg.GetMysqlPort(), cfg.GetMysqlDatabase()))
	if err != nil {
		logger.Critical(ctx, "mysql conn error: %v\n", err)
		os.Exit(1)
	}

	db.SetConnMaxLifetime(cfg.GetMysqlConnMaxLifetime())
	db.SetMaxIdleConns(cfg.GetMysqlMaxIdleConns())
	db.SetMaxOpenConns(cfg.GetMysqlMaxOpenConns())

	return db
}
