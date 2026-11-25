package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
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
		return nil, observability.DatabaseError(err, "database_connect")
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(45 * time.Minute)
	deadline := time.NewTimer(65 * time.Second)
	defer deadline.Stop()
	started := time.Now()
	attempts := 0
	for {
		attempts++
		pingCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			slog.InfoContext(ctx, "database.connected", "component", "database", "operation", "database_connect", "attempts", attempts, "duration_ms", time.Since(started).Milliseconds())
			return db, nil
		}
		if attempts == 1 || attempts%10 == 0 {
			slog.WarnContext(ctx, "database.connection.retry", append([]any{"component", "database", "operation", "database_ping", "attempt", attempts, "retry_planned", true}, observability.DiagnosticFields(err, observability.Source(0))...)...)
		}
		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, ctx.Err()
		case <-deadline.C:
			_ = db.Close()
			return nil, observability.DatabaseError(err, "database_ping")
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
	// Renaming a database does not rename its files. Give every new database
	// incarnation fresh filenames, leaving existing and orphaned files intact.
	const statement = `IF DB_ID(@p1) IS NULL
BEGIN
    DECLARE @dataPath nvarchar(4000) = CONVERT(nvarchar(4000), SERVERPROPERTY(N'InstanceDefaultDataPath'));
    DECLARE @logPath nvarchar(4000) = CONVERT(nvarchar(4000), SERVERPROPERTY(N'InstanceDefaultLogPath'));
    IF NULLIF(@dataPath, N'') IS NULL OR NULLIF(@logPath, N'') IS NULL
        THROW 50000, 'SQL Server default database directories are unavailable', 1;
    IF RIGHT(@dataPath, 1) NOT IN (N'/', N'\')
        SET @dataPath += CASE WHEN CHARINDEX(N'\', @dataPath) > 0 THEN N'\' ELSE N'/' END;
    IF RIGHT(@logPath, 1) NOT IN (N'/', N'\')
        SET @logPath += CASE WHEN CHARINDEX(N'\', @logPath) > 0 THEN N'\' ELSE N'/' END;
    DECLARE @fileBase nvarchar(128) = @p1 + N'_' + REPLACE(CONVERT(nvarchar(36), NEWID()), N'-', N'');
    DECLARE @sql nvarchar(max) = N'CREATE DATABASE ' + QUOTENAME(@p1)
        + N' ON PRIMARY (NAME = ' + QUOTENAME(@p1, N'''')
        + N', FILENAME = N''' + REPLACE(@dataPath + @fileBase + N'.mdf', N'''', N'''''') + N''')'
        + N' LOG ON (NAME = ' + QUOTENAME(@p1 + N'_log', N'''')
        + N', FILENAME = N''' + REPLACE(@logPath + @fileBase + N'_log.ldf', N'''', N'''''') + N''')';
    EXEC sys.sp_executesql @sql;
END`
	_, err = master.ExecContext(ctx, statement, name)
	if err != nil {
		return observability.DatabaseError(err, "database_provision")
	}
	return nil
}
