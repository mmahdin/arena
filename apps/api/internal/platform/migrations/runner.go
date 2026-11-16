package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var migrationName = regexp.MustCompile(`^[0-9]{4}_[a-z0-9_]+\.sql$`)

// Apply runs immutable, SHA256-checked SQL Server migrations in lexical order.
// It uses a connection-scoped SQL Server application lock plus one transaction per file.
// Files must not contain GO, BEGIN TRAN, COMMIT or ROLLBACK statements.
func Apply(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if !migrationName.MatchString(entry.Name()) {
			return fmt.Errorf("invalid migration filename: %s", entry.Name())
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no SQL migrations in %s", dir)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var lockResult int
	err = conn.QueryRowContext(ctx, `DECLARE @lockResult int;
      EXEC @lockResult = sys.sp_getapplock @Resource=N'wellbeing_migrations', @LockMode=N'Exclusive',
        @LockOwner=N'Session', @LockTimeout=15000;
      SELECT @lockResult;`).Scan(&lockResult)
	if err != nil {
		return fmt.Errorf("acquire migration app lock: %w", err)
	}
	if lockResult < 0 {
		return fmt.Errorf("SQL Server app lock failure code %d", lockResult)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(),
			`EXEC sys.sp_releaseapplock @Resource=N'wellbeing_migrations',@LockOwner=N'Session'`)
	}()
	_, err = conn.ExecContext(ctx, `IF OBJECT_ID(N'dbo.schema_migrations',N'U') IS NULL
       CREATE TABLE dbo.schema_migrations (
           name nvarchar(255) NOT NULL PRIMARY KEY,
           sha256 varchar(64) NOT NULL,
           applied_at datetime2(3) NOT NULL DEFAULT SYSUTCDATETIME()
       )`)
	if err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		upper := strings.ToUpper(string(raw))
		if regexp.MustCompile(`(?m)^\s*GO\s*$`).MatchString(upper) {
			return fmt.Errorf("GO batch separator is forbidden: %s", name)
		}
		if regexp.MustCompile(`(?i)\b(BEGIN\s+TRAN(SACTION)?|COMMIT\s*(TRAN(SACTION)?)?|ROLLBACK\s*(TRAN(SACTION)?)?)\b`).MatchString(upper) {
			return fmt.Errorf("migration must not control transactions: %s", name)
		}
		digest := sha256.Sum256(raw)
		fingerprint := hex.EncodeToString(digest[:])
		var existing string
		err = conn.QueryRowContext(ctx, `SELECT sha256 FROM dbo.schema_migrations WHERE name=@p1`, name).Scan(&existing)
		if err == nil {
			if existing != fingerprint {
				return fmt.Errorf("applied migration changed: %s", name)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(raw)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.schema_migrations(name,sha256) VALUES(@p1,@p2)`, name, fingerprint); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("migration %s commit: %w", name, err)
		}
	}
	return nil
}
