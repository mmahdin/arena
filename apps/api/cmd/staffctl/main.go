// Command staffctl is a local-only bootstrap tool for explicitly allowlisting
// existing Keycloak staff accounts. It never creates Keycloak users or passwords.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
)

func localIssuer(environment, base string) (string, error) {
	if environment != "local" {
		return "", errors.New("staffctl is limited to APP_ENV=local")
	}
	if base == "" {
		base = "http://auth.localhost:8180"
	}
	u, err := url.Parse(base)
	if err != nil || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid local Keycloak base URL")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && !strings.HasSuffix(host, ".localhost") && (ip == nil || !ip.IsLoopback()) {
		return "", errors.New("local staff provisioning cannot target a non-loopback Keycloak issuer")
	}
	return base + "/realms/theia-staff", nil
}

func run(args []string, getenv func(string) string, out func(string, ...any)) error {
	flags := flag.NewFlagSet("staffctl", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	action := flags.String("action", "status", "status | provision | deactivate | reactivate")
	subject := flags.String("subject", "", "Keycloak staff realm user ID (UUID, not email or username)")
	confirm := flags.Bool("confirm", false, "explicitly authorize a state-changing operation")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	issuer, err := localIssuer(getenv("APP_ENV"), getenv("AUTH_KEYCLOAK_BASE_URL"))
	if err != nil {
		return err
	}
	if err := mssql.ValidateStaffBinding(issuer, *subject); err != nil {
		return errors.New("-subject must be a canonical Keycloak user UUID from theia-staff realm")
	}
	switch *action {
	case "status":
		if *confirm {
			return errors.New("-confirm is unnecessary for status")
		}
	case "provision", "deactivate", "reactivate":
		if !*confirm {
			return errors.New("mutations require explicit -confirm")
		}
	default:
		return errors.New("unsupported -action")
	}
	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		return errors.New("DATABASE_URL not configured")
	}
	// This bootstrap is deliberately restricted to local SQL Server as well
	// as local Keycloak. Never use it with an accessible remote production DB.
	dbURL, err := url.Parse(databaseURL)
	if err != nil || dbURL.Scheme != "sqlserver" {
		return errors.New("invalid DATABASE_URL")
	}
	dbHost := dbURL.Hostname()
	dbIP := net.ParseIP(dbHost)
	if dbHost != "localhost" && !strings.HasSuffix(dbHost, ".localhost") && (dbIP == nil || !dbIP.IsLoopback()) && dbHost != "mssql" {
		return errors.New("staffctl cannot target a remote SQL Server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := database.Connect(ctx, databaseURL)
	if err != nil {
		return errors.New("SQL Server connection unavailable")
	}
	defer db.Close()
	repo := mssql.NewStaffProvisioner(db)
	switch *action {
	case "status":
		item, err := repo.State(ctx, issuer, *subject)
		if err != nil {
			return errors.New("staff mapping not found or unavailable")
		}
		out("staff mapping exists; staffId=%s active=%t", item.ID, item.Active)
	case "provision":
		item, created, err := repo.Provision(ctx, issuer, *subject)
		if errors.Is(err, auth.ErrInactive) {
			return errors.New("staff is inactive; verify Keycloak before -action=reactivate")
		}
		if err != nil {
			return errors.New("provisioning failed; inspect migration and Keycloak ID")
		}
		out("staff mapping provisioned=%t staffId=%s active=%t", created, item.ID, item.Active)
	case "deactivate", "reactivate":
		item, changed, err := repo.SetActive(ctx, issuer, *subject, *action == "reactivate")
		if err != nil {
			return errors.New("staff state change failed")
		}
		out("staff state changed=%t staffId=%s active=%t", changed, item.ID, item.Active)
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Getenv, func(format string, args ...any) { fmt.Printf(format+"\n", args...) }); err != nil {
		fmt.Fprintln(os.Stderr, "staffctl:", err)
		os.Exit(1)
	}
}
