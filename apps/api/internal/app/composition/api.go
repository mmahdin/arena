package composition

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/policy"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation"
	convapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
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
func NewAPI(cfg config.Config, db *sql.DB) (http.Handler, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), httpx.Middleware(cfg.AllowedOrigin))
	router.GET("/health/live", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			httpx.Failure(c, 503, "DB_UNAVAILABLE")
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	if !cfg.AllowDevLogin {
		return router, nil
	}
	signer, err := devsecurity.NewSigner(cfg.DevSessionSecret, cfg.CookieSecure)
	if err != nil {
		return nil, err
	}
	if !config.SupportsDevLogin(cfg.Environment) {
		return nil, errors.New("dev login can be enabled only in local or staging environments")
	}
	dev := devsession.NewModule(db, signer, cfg.DevSessionSecret)
	group := router.Group("/api/v1")
	group.GET("/dev/users", dev.Handler.List) // legacy internal-only test route
	group.POST("/dev/identity-session", dev.Handler.IdentityLogin)
	group.POST("/dev/session", dev.Handler.Login)
	group.DELETE("/dev/session", dev.Handler.Logout)
	authorized := group.Group("")
	authorized.Use(devsecurity.Guard(signer, dev.Sessions))
	authorized.GET("/me/ai-status", aiStatus(cfg))
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
	chat := conversation.NewModule(db, proReader, ai.NewModule(cfg), cfg.MaxContextMessages)
	authorized.GET("/conversations", chat.List)
	authorized.POST("/conversations", chat.Create)
	authorized.GET("/conversations/:id/messages", chat.Messages)
	authorized.DELETE("/conversations/:id", chat.Delete)
	authorized.PATCH("/conversations/:id", chat.Rename)
	authorized.POST("/conversations/:id/turns", chat.Turn)
	personalLetters := letters.NewModule(db, cfg)
	authorized.GET("/me/letters", personalLetters.List)
	authorized.POST("/me/letters", personalLetters.Generate)
	authorized.DELETE("/me/letters/:id", personalLetters.Delete)
	reflections := reflection.NewModule(db)
	authorized.GET("/me/daily-reflections", reflections.List)
	authorized.GET("/me/daily-reflections/:date", reflections.Get)
	authorized.PUT("/me/daily-reflections/:date", reflections.Save)
	authorized.DELETE("/me/daily-reflections/:date", reflections.Delete)
	saved := bookmark.NewModule(db)
	authorized.GET("/me/bookmarks", saved.List)
	authorized.PUT("/messages/:messageId/bookmark", saved.Save)
	authorized.DELETE("/messages/:messageId/bookmark", saved.Remove)
	return router, nil
}

// Configuration only: this deliberately makes no billable provider request.
func aiStatus(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var model any
		if cfg.LLMProvider == "openai-compatible" {
			model = cfg.ModelName
		}
		c.JSON(http.StatusOK, gin.H{"provider": cfg.LLMProvider, "model": model, "mock": cfg.LLMProvider == "mock", "promptVersion": policy.Version})
	}
}
