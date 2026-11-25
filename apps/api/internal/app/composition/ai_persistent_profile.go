package composition

import (
	"context"
	"database/sql"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// Cross-module profile projection stays in composition and uses the owning
// User application. No AI adapter queries the User module's tables directly.
type persistentProfileReader struct {
	memories contextstate.PersistentReader
	profiles *userapp.Service
}

func newPersistentProfileReader(memories contextstate.PersistentReader, db *sql.DB) persistentProfileReader {
	return persistentProfileReader{memories: memories, profiles: userapp.New(userdb.New(db))}
}
func (p persistentProfileReader) Persistent(ctx context.Context, user, trigger c.ID) (c.PersistentContext, error) {
	out, err := p.memories.Persistent(ctx, user, trigger)
	if err != nil {
		return out, err
	}
	profile, e := p.profiles.Get(ctx, user)
	if errors.Is(e, httpx.ErrNotFound) {
		return out, nil
	}
	if e != nil {
		out.Degraded = true
		out.Reason = "profile_unavailable"
		return out, nil
	}
	if profile.UserID != user {
		return c.PersistentContext{}, &c.Error{Code: c.ErrForbidden, Step: "persistent_profile"}
	}
	v := &c.ProfileView{DisplayName: profile.DisplayName, ConversationStyle: string(profile.ConversationStyle), Onboarding: map[string][]string{}}
	if profile.OnboardingCompletedAt != nil && profile.OnboardingAnswers != nil {
		a := profile.OnboardingAnswers.ForModel()
		v.Onboarding = map[string][]string{"focus": a.Focus, "concerns": a.Concerns, "relationships": a.Relationships, "strengths": a.Strengths, "recharge": a.Recharge}
	}
	out.Profile = v
	return out, nil
}
