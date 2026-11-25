// Command analytics-reader provisions a dedicated SQL Server reporting login.
// It is LOCAL-ONLY and must run after migration 0037. It never prints secrets.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
)

var generatedSecret = regexp.MustCompile(`^Aa1![0-9a-f]{60}$`)

func validate(env, secret string) error {
	if env != "local" {
		return errors.New("analytics reader provisioning is local-only")
	}
	if !generatedSecret.MatchString(secret) {
		return errors.New("invalid analytics reader secret: expected a locally generated 64-character policy-compliant secret")
	}
	return nil
}

func validateDatabaseEndpoint(dsn string) error {
	endpoint, err := url.Parse(dsn)
	if err != nil || endpoint.Scheme != "sqlserver" {
		return errors.New("invalid database settings")
	}
	host := endpoint.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("analytics reader provisioning requires loopback SQL Server")
	}
	return nil
}

func provision(ctx context.Context, db *sql.DB, secret string) error {
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT DB_NAME()`).Scan(&databaseName); err != nil {
		return errors.New("cannot inspect database")
	}
	if databaseName != "wellbeing" {
		return errors.New("refusing to provision outside local wellbeing database")
	}
	var migrated int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.schema_migrations WHERE name=N'0037_product_reporting_views.sql'`).Scan(&migrated); err != nil || migrated != 1 {
		return errors.New("migration 0037 must be applied first")
	}
	// Allow the CONNECT SQL grant created by SQL Server for every login.
	// Other direct grants or server roles could make this credential an
	// unintended privileged login. This check runs BEFORE password changes.
	var elevated int
	if err := db.QueryRowContext(ctx, `SELECT
        (SELECT COUNT(*) FROM sys.server_principals
         WHERE name=N'theia_metabase_reader' AND type<>'S')
        + (SELECT COUNT(*) FROM sys.server_role_members
         WHERE member_principal_id=SUSER_ID(N'theia_metabase_reader'))
        + (SELECT COUNT(*) FROM sys.server_permissions
           WHERE grantee_principal_id=SUSER_ID(N'theia_metabase_reader')
             AND (state='W' OR (state='G' AND permission_name<>N'CONNECT SQL')))
        + (SELECT COUNT(*) FROM sys.databases
           WHERE owner_sid=SUSER_SID(N'theia_metabase_reader'))`).Scan(&elevated); err != nil || elevated != 0 {
		return errors.New("existing reporting login has elevated server privileges or cannot be verified")
	}
	// Check database privileges before modifying an existing login's
	// password. An orphaned or privileged pre-existing user is not ours.
	var incompatibleUser int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sys.database_principals AS p
      WHERE p.name=N'theia_metabase_reader' AND (
        SUSER_SID(N'theia_metabase_reader') IS NULL
        OR p.sid<>SUSER_SID(N'theia_metabase_reader')
        OR EXISTS (SELECT 1 FROM sys.database_role_members AS m
                   INNER JOIN sys.database_principals AS r ON r.principal_id=m.role_principal_id
                   WHERE m.member_principal_id=p.principal_id
                     AND r.name<>N'theia_analytics_reader')
        OR EXISTS (SELECT 1 FROM sys.database_permissions AS d
                   WHERE d.grantee_principal_id=p.principal_id
                     AND (d.state='W' OR (d.state='G' AND d.permission_name<>N'CONNECT'))))`).Scan(&incompatibleUser); err != nil || incompatibleUser != 0 {
		return errors.New("existing reporting database user is elevated or improperly mapped")
	}
	// CREATE/ALTER LOGIN is server-scope and cannot be parameterized directly.
	// The fixed policy-compliant generated secret has strictly limited syntax.
	statement := fmt.Sprintf(`IF NOT EXISTS (SELECT 1 FROM sys.sql_logins WHERE name=N'theia_metabase_reader')
      EXEC(N'CREATE LOGIN [theia_metabase_reader] WITH PASSWORD=''%s'',CHECK_POLICY=ON,CHECK_EXPIRATION=OFF');
      ELSE EXEC(N'ALTER LOGIN [theia_metabase_reader] WITH PASSWORD=''%s''');`, secret, secret)
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return errors.New("failed to create/update restricted SQL login")
	}
	if _, err := db.ExecContext(ctx, `IF NOT EXISTS (SELECT 1 FROM sys.database_principals WHERE name=N'theia_metabase_reader')
      CREATE USER [theia_metabase_reader] FOR LOGIN [theia_metabase_reader];
      IF EXISTS (SELECT 1 FROM sys.database_principals
                 WHERE name=N'theia_metabase_reader' AND sid<>SUSER_SID(N'theia_metabase_reader'))
        THROW 51061,'Reporting user is mapped to the wrong SQL login',1;
      IF EXISTS (SELECT 1 FROM sys.database_role_members AS m
                 INNER JOIN sys.database_principals AS r ON r.principal_id=m.role_principal_id
                 WHERE m.member_principal_id=USER_ID(N'theia_metabase_reader')
                   AND r.name<>N'theia_analytics_reader')
        THROW 51062,'Reporting user has unrelated database roles',1;
      IF EXISTS (SELECT 1 FROM sys.database_permissions
                 WHERE grantee_principal_id=USER_ID(N'theia_metabase_reader')
                   AND (state='W' OR (state='G' AND permission_name<>N'CONNECT')))
        THROW 51063,'Reporting user has extra database grants',1;
      IF ISNULL(IS_ROLEMEMBER(N'theia_analytics_reader',N'theia_metabase_reader'),0)<>1
        ALTER ROLE [theia_analytics_reader] ADD MEMBER [theia_metabase_reader];`); err != nil {
		return errors.New("reporting account is elevated, misconfigured, or cannot be provisioned")
	}
	return nil
}

func main() {
	secret := os.Getenv("THEIA_METABASE_READER_PASSWORD")
	if err := validate(os.Getenv("APP_ENV"), secret); err != nil {
		log.Fatal(err)
	}
	dsn, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal("invalid database settings")
	}
	if err := validateDatabaseEndpoint(dsn); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		log.Fatal("cannot connect to local SQL Server")
	}
	defer db.Close()
	if err := provision(ctx, db, secret); err != nil {
		log.Fatal(err)
	}
	log.Print("local analytics SQL reader provisioned (view-only, secret not displayed)")
}
