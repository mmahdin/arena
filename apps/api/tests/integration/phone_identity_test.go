package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/app/composition"
	devdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/adapters/mssql"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	userdomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Disposable SQL Server database only; no existing developer data is overwritten.
func TestSyntheticPhoneIdentitySQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("requires local SQL Server integration opt-in")
	}
	raw := os.Getenv("DATABASE_URL")
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("integration requires local SQL Server")
	}
	if host := strings.ToLower(parsed.Hostname()); host != "127.0.0.1" && host != "localhost" && host != "mssql" {
		t.Fatal("integration requires local SQL Server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	name := "wellbeing_phone_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	query := parsed.Query()
	query.Set("database", name)
	parsed.RawQuery = query.Encode()
	if err := database.EnsureLocalDatabase(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	query.Set("database", "master")
	masterURL := *parsed
	masterURL.RawQuery = query.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, err := master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); err != nil {
			t.Error(err)
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Upgrade a populated pre-0008 database and compare every original table.
	previous := t.TempDir()
	files, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "0008_" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(previous, filepath.Base(file)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Apply(ctx, db, previous); err != nil {
		t.Fatal(err)
	}
	fingerprint := func() [4][2]int64 {
		t.Helper()
		var result [4][2]int64
		for i, table := range []string{"users", "user_profiles", "conversations", "messages"} {
			columns := "*"
			if table == "user_profiles" {
				columns = "user_id,display_name,conversation_style,onboarding_completed_at,updated_at"
			}
			if err := db.QueryRowContext(ctx, "SELECT COUNT_BIG(*),COALESCE(CHECKSUM_AGG(BINARY_CHECKSUM("+columns+")),0) FROM dbo."+table).Scan(&result[i][0], &result[i][1]); err != nil {
				t.Fatal(err)
			}
		}
		return result
	}
	before := fingerprint()
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if after := fingerprint(); after != before {
		t.Fatal("upgrade changed existing column values")
	}
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	repo := devdb.New(db, "synthetic-secret-with-at-least-32-characters")
	newSession := func(phone string) (uuid.UUID, bool, error) {
		return repo.FindOrCreatePhoneSession(ctx, phone, uuid.New(), time.Now().Add(time.Hour))
	}
	first, created, err := newSession("09000000001")
	if err != nil || !created || first == uuid.Nil {
		t.Fatalf("first create: created=%v err=%v", created, err)
	}
	again, created, err := newSession("09000000001")
	if err != nil || created || first != again {
		t.Fatalf("re-entry did not resolve same user: created=%v err=%v", created, err)
	}
	var unfinished int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.user_profiles WHERE user_id=CONVERT(uniqueidentifier,@p1) AND onboarding_completed_at IS NULL`, first.String()).Scan(&unfinished); err != nil || unfinished != 1 {
		t.Fatalf("missing initial incomplete profile: %v", err)
	}
	var rawStored int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.users WHERE test_key LIKE N'%09000000001%'`).Scan(&rawStored); err != nil || rawStored != 0 {
		t.Fatal("raw synthetic phone stored")
	}
	updated, err := userapp.New(userdb.New(db)).Update(ctx, first, "کاربر تستی", userdomain.Balanced)
	if err != nil || updated.OnboardingCompletedAt == nil {
		t.Fatalf("onboarding update: %v", err)
	}
	returning, wasNew, err := newSession("09000000001")
	if err != nil || wasNew || returning != first {
		t.Fatal("returning user mismatch")
	}
	loaded, err := userapp.New(userdb.New(db)).Get(ctx, returning)
	if err != nil || loaded.OnboardingCompletedAt == nil || loaded.DisplayName != "کاربر تستی" {
		t.Fatal("completed onboarding not retained")
	}
	const clients = 6
	var wg sync.WaitGroup
	ids := make(chan uuid.UUID, clients)
	errs := make(chan error, clients)
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); id, _, err := newSession("09000000002"); ids <- id; errs <- err }()
	}
	wg.Wait()
	close(ids)
	close(errs)
	var expected uuid.UUID
	for id := range ids {
		if id == uuid.Nil {
			t.Fatal("nil concurrent identity")
		}
		if expected == uuid.Nil {
			expected = id
		}
		if expected != id {
			t.Fatal("duplicate identities created")
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var links int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.dev_phone_identities`).Scan(&links); err != nil || links != 2 {
		t.Fatalf("expected exactly two synthetic identities, got %d: %v", links, err)
	}
	t.Run("identity and session transaction rolls back together", func(t *testing.T) {
		sid := uuid.New()
		if _, _, err := repo.FindOrCreatePhoneSession(ctx, "09000000001", sid, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		// A duplicate session PK must roll back the newly inserted user/profile/mapping.
		before := fingerprint()
		if _, _, err := repo.FindOrCreatePhoneSession(ctx, "09000000003", sid, time.Now().Add(time.Hour)); err == nil {
			t.Fatal("duplicate session accepted")
		}
		if fingerprint() != before {
			t.Fatal("failed session left identity/profile rows")
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dbo.dev_phone_identities").Scan(&count); err != nil || count != 2 {
			t.Fatal("failed transaction left a mapping")
		}
	})
	t.Run("dynamic identity over real HTTP", func(t *testing.T) { phoneIdentityHTTP(t, db) })
}

func phoneIdentityHTTP(t *testing.T, db *sql.DB) {
	t.Helper()
	cfg := config.Config{Environment: "local", AllowDevLogin: true, DevSessionSecret: "synthetic-secret-with-at-least-32-characters", AllowedOrigin: "http://127.0.0.1:5173", LLMProvider: "mock", MaxContextMessages: 20, MaxContextChars: 16000}
	handler, err := composition.NewAPI(cfg, db)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	t.Run("concurrent first identity HTTP requests", func(t *testing.T) {
		const clients = 8
		type result struct {
			id      string
			created bool
			err     error
			status  int
		}
		results := make(chan result, clients)
		for i := 0; i < clients; i++ {
			go func() {
				client, _ := newAcceptanceClient(t)
				response, err := client.Post(server.URL+"/api/v1/dev/identity-session", "application/json", bytes.NewBufferString(`{"phone":"09000000006"}`))
				if err != nil {
					results <- result{err: err}
					return
				}
				var login struct {
					Created bool `json:"created"`
				}
				err = json.NewDecoder(response.Body).Decode(&login)
				response.Body.Close()
				if err != nil || response.StatusCode != http.StatusOK {
					results <- result{err: err, status: response.StatusCode}
					return
				}
				me, err := client.Get(server.URL + "/api/v1/me")
				if err != nil {
					results <- result{err: err}
					return
				}
				defer me.Body.Close()
				var profile struct {
					ID string `json:"userId"`
				}
				err = json.NewDecoder(me.Body).Decode(&profile)
				results <- result{id: profile.ID, created: login.Created, err: err, status: me.StatusCode}
			}()
		}
		var id string
		created := 0
		for i := 0; i < clients; i++ {
			result := <-results
			if result.err != nil || result.status != 200 || result.id == "" {
				t.Fatalf("concurrent HTTP identity status %d error %v", result.status, result.err)
			}
			if id == "" {
				id = result.id
			}
			if result.id != id {
				t.Fatal("concurrent HTTP duplicate identity")
			}
			if result.created {
				created++
			}
		}
		if created != 1 {
			t.Fatalf("created accounts = %d, want 1", created)
		}
	})
	client, jar := newAcceptanceClient(t)
	login := func(phone string, created bool) string {
		t.Helper()
		result := acceptRequest(t, client, server.URL, "POST", "/api/v1/dev/identity-session", map[string]any{"phone": phone}, 200)
		if result["created"] != created {
			t.Fatal("unexpected created flag")
		}
		return acceptString(t, acceptRequest(t, client, server.URL, "GET", "/api/v1/me", nil, 200), "userId")
	}
	a := login("۰۹۰۰۰۰۰۰۰۰۴", true)
	cookies := jar.Cookies(mustURL(t, server.URL+"/api/v1/me"))
	if len(cookies) != 1 {
		t.Fatal("missing session cookie")
	}
	old := cookies[0].Value
	if a != login("٠٩٠٠٠٠٠٠٠٠٤", false) || a != login(" 09000000004 ", false) {
		t.Fatal("normalization changed identity")
	}
	assertAcceptanceCode(t, acceptanceWithStolenCookie(t, server.URL, old), "SESSION_INVALID")
	for _, phone := range []string{"", "0900000000", "090000000044", "+9809000000004", "0900000x004", "0900000 004"} {
		assertAcceptanceCode(t, acceptRequest(t, client, server.URL, "POST", "/api/v1/dev/identity-session", map[string]any{"phone": phone}, 400), "INVALID_DEV_PHONE")
	}
	acceptRequest(t, client, server.URL, "POST", "/api/v1/conversations", nil, 409)
	acceptRequest(t, client, server.URL, "PATCH", "/api/v1/me/profile", map[string]any{"displayName": "", "conversationStyle": "balanced"}, 400)
	profile := acceptRequest(t, client, server.URL, "PATCH", "/api/v1/me/profile", map[string]any{"displayName": "آزمایش ساختگی", "conversationStyle": "listening"}, 200)
	conv := acceptString(t, acceptRequest(t, client, server.URL, "POST", "/api/v1/conversations", nil, 201), "id")
	route := "/api/v1/conversations/" + conv
	body := map[string]any{"clientTurnId": uuid.NewString(), "message": "پیام ساختگی برای آزمون"}
	acceptRequest(t, client, server.URL, "POST", route+"/turns", body, 200)
	acceptRequest(t, client, server.URL, "POST", route+"/turns", body, 200)
	assertMessages(t, acceptRequest(t, client, server.URL, "GET", route+"/messages", nil, 200), 2)
	old = jar.Cookies(mustURL(t, server.URL+"/api/v1/me"))[0].Value
	b := login("09000000005", true)
	if a == b {
		t.Fatal("identities mixed")
	}
	assertAcceptanceCode(t, acceptanceWithStolenCookie(t, server.URL, old), "SESSION_INVALID")
	for _, method := range []string{"GET", "DELETE"} {
		path := route
		if method == "GET" {
			path += "/messages"
		}
		acceptRequest(t, client, server.URL, method, path, nil, 404)
	}
	acceptRequest(t, client, server.URL, "POST", route+"/turns", body, 404)
	// Client-supplied identity cannot choose whose profile is edited.
	own := acceptRequest(t, client, server.URL, "PATCH", "/api/v1/me/profile", map[string]any{"userId": a, "displayName": "آزمایش", "conversationStyle": "balanced"}, 200)
	if own["userId"] != b {
		t.Fatal("client userId changed profile ownership")
	}
	if login("09000000004", false) != a {
		t.Fatal("returning identity changed")
	}
	loaded := acceptRequest(t, client, server.URL, "GET", "/api/v1/me", nil, 200)
	if loaded["onboardingCompletedAt"] != profile["onboardingCompletedAt"] || loaded["displayName"] != profile["displayName"] {
		t.Fatal("profile lost after switch")
	}
	assertMessages(t, acceptRequest(t, client, server.URL, "GET", route+"/messages", nil, 200), 2)
	old = jar.Cookies(mustURL(t, server.URL+"/api/v1/me"))[0].Value
	acceptRequest(t, client, server.URL, "DELETE", "/api/v1/dev/session", nil, 204)
	assertAcceptanceCode(t, acceptanceWithStolenCookie(t, server.URL, old), "SESSION_INVALID")
	if login("09000000004", false) != a {
		t.Fatal("logout lost identity")
	}
	acceptRequest(t, client, server.URL, "DELETE", route, nil, 204)
}
