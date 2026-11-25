package composition

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/product"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation"
	convapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
	"log/slog"
	"net/http"
	"time"
)

type profileReader struct{ service *userapp.Service }

func (p profileReader) ForUser(ctx context.Context, id uuid.UUID) (convapp.Profile, error) {
	data, err := p.service.Get(ctx, id)
	profile := convapp.Profile{DisplayName: data.DisplayName, ConversationStyle: string(data.ConversationStyle), OnboardingComplete: data.OnboardingCompletedAt != nil}
	if data.OnboardingAnswers != nil {
		answers := convapp.OnboardingAnswers(data.OnboardingAnswers.ForModel())
		profile.OnboardingAnswers = &answers
	}
	return profile, err
}
func NewAPI(cfg config.Config, db *sql.DB, options ...ProductOptions) (*API, error) {
	var localInspector *trace.Inspector
	if cfg.LocalInspector {
		var err error
		localInspector, err = trace.NewInspector(trace.InspectorOptions{Directory: cfg.LocalInspectorDir, Environment: cfg.Environment, Address: cfg.Address, OnError: func(error) { slog.Warn("ai.local_inspector.write_failed") }})
		if err != nil {
			return nil, err
		}
		localInspector.RegisterSecret(cfg.ModelAPIKey)
	}

	api := &API{otel: cfg.OTelRuntime}
	// One metadata-only writer shared by active User turns and claimed workers.
	// No mapping is attempted when OTel or SQL is unavailable.
	if cfg.OTelRuntime != nil && db != nil {
		api.traceRefWriter = aidb.NewTraceReferenceRepository(db)
	}
	opt := ProductOptions{}
	if len(options) > 1 {
		return nil, ErrRuntime
	}
	if len(options) == 1 {
		opt = options[0]
	}
	if cfg.Auth.BrowserSmokeEnabled && !config.BrowserSmokeAllowed(cfg.Auth, cfg.Environment) {
		return nil, errors.New("browser auth smoke requires local BFF and loopback origins")
	}
	// Even manually constructed configs may not enable an unreviewed bundle on a public listener.
	if cfg.LocalKBDraft && (cfg.Environment != "local" || cfg.AICoreConfigFile == "" || !config.LocalDraftListenerAllowed(cfg.Address)) {
		return nil, ErrRuntime
	}
	var bff *authhttp.Handler
	if cfg.Auth.Enabled {
		if db == nil {
			return nil, errors.New("BFF requires SQL Server")
		}
		var e error
		bff, e = authhttp.New(context.Background(), db, cfg.Auth, cfg.Environment == "local")
		if e != nil {
			return nil, e
		}
	}
	// AI Core deployment is independent of browser authentication.
	if len(options) == 0 && cfg.AICoreConfigFile != "" {
		var e error
		opt, e = loadProduct(metrics.WithRuntime(trace.WithInspector(context.Background(), localInspector), cfg.MetricsRuntime), db, cfg.AICoreConfigFile, cfg.LocalKBDraft)
		if e != nil {
			return nil, e
		}
	}
	if opt.Core != nil && (opt.Core.Turns == nil || opt.Core.Memory == nil || opt.Core.MemoryWorker == nil || opt.Core.Reflection == nil || opt.Core.Scheduler == nil) {
		return nil, ErrRuntime
	}
	if opt.Core != nil {
		copy := *opt.Core
		copy.Scheduler = opt.Core.Scheduler.WithoutDispatch()
		if cfg.MetricsRuntime != nil {
			cfg.MetricsRuntime.ExpectWorkers()
			copy.Pulse = meteredWorkerPulse{next: copy.Pulse, meter: cfg.MetricsRuntime}
		}
		api.core = &copy
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	// Attach an in-process tracer only. Do not extract untrusted browser
	// traceparent/baggage headers and do not create a new HTTP public route.
	if cfg.OTelRuntime != nil || api.core != nil {
		router.Use(func(c *gin.Context) {
			ctx := c.Request.Context()
			if cfg.OTelRuntime != nil {
				ctx = telemetry.WithRuntime(ctx, cfg.OTelRuntime)
			}
			if api.core != nil {
				ctx = trace.WithTimeouts(ctx, trace.Timeouts{ReferenceWrite: time.Duration(api.core.Budgets.Operations.TraceReferenceMS) * time.Millisecond, JobRefresh: time.Duration(api.core.Budgets.Operations.TraceJobsMS) * time.Millisecond})
			}
			c.Request = c.Request.WithContext(trace.WithReferenceWriter(ctx, api.traceRefWriter))
			c.Next()
		})
	}
	router.Use(httpx.RequestLog(slog.Default()))
	if cfg.MetricsRuntime != nil {
		router.Use(cfg.MetricsRuntime.Middleware())
	}
	router.Use(httpx.Recovery(slog.Default()), httpx.Middleware(cfg.AllowedOrigin))
	api.Handler = router
	router.NoRoute(func(c *gin.Context) { httpx.Failure(c, 404, "RESOURCE_NOT_FOUND") })
	router.GET("/health/live", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if db == nil {
			httpx.Failure(c, 503, "DB_UNAVAILABLE")
			return
		}
		if err := db.PingContext(ctx); err != nil {
			httpx.FailureWithError(c, 503, "DB_UNAVAILABLE", err)
			return
		}
		// Readiness reports AI worker availability regardless of login mechanism.
		if api.core == nil || !api.healthy.Load() {
			httpx.Failure(c, 503, "AI_CORE_UNAVAILABLE")
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	// Real browser authentication is opt-in. Both login surfaces share this
	// module but never share OIDC realms, cookies or host bindings.
	if bff == nil {
		pendingAuth := func(c *gin.Context) { httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_NOT_CONFIGURED") }
		router.GET("/auth/user/login", pendingAuth)
		router.GET("/auth/user/callback", pendingAuth)
		router.GET("/auth/user/session", pendingAuth)
		router.POST("/auth/user/logout", pendingAuth)
		router.POST("/auth/user/logout/sso", pendingAuth)
		router.POST("/auth/user/logout-all", pendingAuth)
		router.POST("/auth/user/backchannel-logout", pendingAuth)
		router.GET("/auth/staff/login", pendingAuth)
		router.GET("/auth/staff/callback", pendingAuth)
		router.GET("/auth/staff/session", pendingAuth)
		router.POST("/auth/staff/logout", pendingAuth)
		router.POST("/auth/staff/logout/sso", pendingAuth)
		router.POST("/auth/staff/logout-all", pendingAuth)
		router.POST("/auth/staff/backchannel-logout", pendingAuth)
	} else {
		router.GET("/auth/user/login", bff.Login(auth.User))
		router.GET("/auth/user/callback", bff.Callback(auth.User))
		router.GET("/auth/user/session", bff.Session(auth.User))
		router.POST("/auth/user/logout", bff.Logout(auth.User))
		router.POST("/auth/user/logout/sso", bff.LogoutWithSSO(auth.User, false))
		router.POST("/auth/user/logout-all", bff.LogoutWithSSO(auth.User, true))
		router.POST("/auth/user/backchannel-logout", bff.Backchannel(auth.User))
		router.GET("/auth/staff/login", bff.Login(auth.Staff))
		router.GET("/auth/staff/callback", bff.Callback(auth.Staff))
		router.GET("/auth/staff/session", bff.Session(auth.Staff))
		router.POST("/auth/staff/logout", bff.Logout(auth.Staff))
		router.POST("/auth/staff/logout/sso", bff.LogoutWithSSO(auth.Staff, false))
		router.POST("/auth/staff/logout-all", bff.LogoutWithSSO(auth.Staff, true))
		router.POST("/auth/staff/backchannel-logout", bff.Backchannel(auth.Staff))
		// Deliberately absent unless explicitly enabled on local loopback
		// origins. No dev/identity bypass or operational data route is added.
		if cfg.Auth.BrowserSmokeEnabled {
			router.GET("/__auth-smoke/user", bff.SmokePage(auth.User))
			router.GET("/__auth-smoke/staff", bff.SmokePage(auth.Staff))
			router.GET("/__auth-smoke/smoke.js", bff.SmokeJavaScript())
		}
	}

	group := router.Group("/api/v1")
	// Admin has an independent guarded route group and never inherits User identity.
	adminAPI := group.Group("/admin")
	if bff == nil {
		adminAPI.Use(func(c *gin.Context) {
			httpx.Failure(c, http.StatusServiceUnavailable, "AUTH_NOT_CONFIGURED")
			c.Abort()
		})
	} else {
		adminAPI.Use(bff.RequireStaff())
	}
	adminModule := admin.NewModule(db, opt.BundleInventory)
	adminAPI.GET("/me", adminModule.Me)
	adminAPI.GET("/audit", adminModule.Audit)
	adminAPI.GET("/overview", adminModule.Overview)
	adminAPI.GET("/turns", adminModule.Turns)
	adminAPI.GET("/turns/:turnId", adminModule.TurnDetail)
	adminAPI.GET("/turns/:turnId/trace-references", adminModule.TraceReferences)
	adminAPI.GET("/jobs", adminModule.Jobs)
	adminAPI.GET("/workers", adminModule.Workers)
	adminAPI.GET("/bundles", adminModule.Bundles)
	adminAPI.GET("/evaluations", adminModule.Evaluations)
	adminAPI.GET("/evaluations/:runId", adminModule.EvaluationDetail)
	adminAPI.POST("/evaluations/import", adminModule.EvaluationImport)

	// AUTH-09: all product routes require the real User BFF. No synthetic
	// login routes, unverified identity selection or legacy cookie fallback.
	userGuard := func(c *gin.Context) {
		httpx.Failure(c, http.StatusUnauthorized, "SESSION_REQUIRED")
		c.Abort()
	}
	if bff != nil && cfg.Auth.UserLoginEnabled {
		userGuard = bff.RequireUser()
	}
	authorized := group.Group("")
	authorized.Use(userGuard)
	profile := user.NewModule(db)
	authorized.GET("/me", profile.Get)
	authorized.PATCH("/me/profile", profile.Update)
	authorized.PATCH("/me/onboarding-name", profile.SaveOnboardingName)
	privacy := user.NewPrivacyModule(db)
	authorized.GET("/me/privacy-preferences", privacy.Get)
	authorized.PUT("/me/privacy-preferences", privacy.Set)
	cursor := user.NewProgressModule(db)
	authorized.GET("/me/onboarding-progress", cursor.Get)
	authorized.PUT("/me/onboarding-progress", cursor.Advance)
	// Explicit port adapter: conversation consumes only a read-only profile interface.
	proReader := profileReader{service: userapp.New(userdb.New(db))}
	chat := conversation.NewModule(db, proReader)
	var ps *product.Service
	if api.core != nil {
		var e error
		ps, e = product.New(api.core.Turns, api.core.Memory, aidb.NewProductRepository(db, NewAIPersistence(db, time.Duration(api.core.Budgets.Operations.TurnLeaseMS)*time.Millisecond)), userdb.New(db), product.Options{ConsentKeys: opt.ConsentKeys, ChatConsentKey: opt.ChatConsentKey, EmbeddingInvalidator: api.core.Embeddings})
		if e != nil {
			return nil, e
		}
	}
	coreHTTP := ai.NewProductHTTP(ps)
	authorized.GET("/conversations", chat.List)
	authorized.POST("/conversations", chat.Create)
	authorized.GET("/conversations/:id/messages", chat.Messages)
	authorized.DELETE("/conversations/:id", chat.Delete)
	authorized.PATCH("/conversations/:id", chat.Rename)
	authorized.POST("/conversations/:id/turns", coreHTTP.Turn)
	authorized.GET("/conversations/:id/turns/:clientTurnId", coreHTTP.Status)
	authorized.POST("/conversations/:id/turns/:clientTurnId/retry", coreHTTP.Retry)
	authorized.POST("/me/ai/recover", coreHTTP.Recover)
	authorized.GET("/me/ai/controls", coreHTTP.Controls)
	authorized.PATCH("/me/ai/controls", coreHTTP.SetControls)
	authorized.GET("/me/memories", coreHTTP.Memories)
	authorized.DELETE("/me/memories", coreHTTP.HideMemories)
	authorized.PATCH("/me/memories/:memoryId", coreHTTP.EditMemory)
	authorized.DELETE("/me/memories/:memoryId", coreHTTP.HideMemory)
	authorized.POST("/me/memories/:memoryId/corrections", coreHTTP.CorrectMemory)
	authorized.DELETE("/messages/:messageId", coreHTTP.HideMessage)
	authorized.GET("/me/ai/export", coreHTTP.Export)
	authorized.DELETE("/me", coreHTTP.HideAccount)
	personalLetters := letters.NewModule(db, cfg)
	authorized.GET("/me/letters", personalLetters.List)
	authorized.POST("/me/letters", personalLetters.Generate)
	authorized.DELETE("/me/letters/:id", personalLetters.Delete)
	saved := bookmark.NewModule(db)
	authorized.GET("/me/bookmarks", saved.List)
	authorized.PUT("/messages/:messageId/bookmark", saved.Save)
	authorized.DELETE("/messages/:messageId/bookmark", saved.Remove)
	return api, nil
}
