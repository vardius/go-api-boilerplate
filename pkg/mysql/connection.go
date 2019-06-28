package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/vardius/golog"
)

// NewConnection provides new mysql connection
func NewConnection(ctx context.Context, host string, port int, user, pass, database string, logger golog.Logger) (db *sql.DB) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, pass, host, port, database))
	if err != nil {
		logger.Critical(ctx, "mysql conn error: %v\n", err)
		os.Exit(1)
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(5)

	return db
}
