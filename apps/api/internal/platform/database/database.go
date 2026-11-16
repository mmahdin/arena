package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

// Connect checks that SQL Server is reachable. It retries during local Docker startup.
// DATABASE_URL must use the sqlserver:// URL format, never a PostgreSQL DSN.
func Connect(ctx context.Context, dsn string) (*sql.DB, error) {
	if !strings.HasPrefix(dsn, "sqlserver://") {
		return nil, errors.New("DATABASE_URL must start with sqlserver://")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQL Server: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(45 * time.Minute)
	deadline := time.NewTimer(65 * time.Second)
	defer deadline.Stop()
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return db, nil
		}
		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, ctx.Err()
		case <-deadline.C:
			_ = db.Close()
			return nil, fmt.Errorf("SQL Server unavailable after startup retry: %w", err)
		case <-time.After(2 * time.Second):
		}
	}
}

var safeDBName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

// EnsureLocalDatabase is only for local synthetic-data bootstrap.
// It connects to master and creates the target DB if necessary. Do not use this as a
// production database provisioning strategy.
func EnsureLocalDatabase(ctx context.Context, dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	name := u.Query().Get("database")
	if !safeDBName.MatchString(name) || name == "master" {
		return errors.New("invalid local database name")
	}
	q := u.Query()
	q.Set("database", "master")
	u.RawQuery = q.Encode()
	master, err := Connect(ctx, u.String())
	if err != nil {
		return err
	}
	defer master.Close()
	statement := "IF DB_ID(N'" + name + "') IS NULL EXEC(N'CREATE DATABASE [" + name + "]')"
	_, err = master.ExecContext(ctx, statement)
	if err != nil {
		return fmt.Errorf("provision local database: %w", err)
	}
	return nil
}
