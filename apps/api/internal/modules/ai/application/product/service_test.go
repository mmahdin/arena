package product

import (
	"context"
	"errors"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type productRepo struct {
	Repository
	controls      Controls
	controlsErr   error
	controlsCalls int
	ownerErr      error
	readyErr      error
	ownerCalls    int
	turnReads     int
}

func (r *productRepo) ValidateOwner(context.Context, c.ID, c.ID) error {
	r.ownerCalls++
	return r.ownerErr
}
func (r *productRepo) Ready(context.Context, c.ID) error { return r.readyErr }
func (r *productRepo) Controls(context.Context, c.ID) (Controls, error) {
	r.controlsCalls++
	return r.controls, r.controlsErr
}
func (r *productRepo) TurnInConversation(context.Context, c.ID, c.ID, c.ID) error {
	return r.ownerErr
}
func (r *productRepo) GetTurn(context.Context, c.ID, c.ID) (c.TurnResult, error) {
	r.turnReads++
	status := c.TurnFailed
	if r.turnReads > 1 {
		status = c.TurnQueued
	}
	return c.TurnResult{Status: status}, nil
}

type productTurns struct {
	Turns
	calls        int
	retryCalls   int
	recoverCalls int
	user         c.ID
	client       c.ID
	req          c.TurnRequest
}

func (t *productTurns) ProcessTurn(_ context.Context, r c.TurnRequest) (c.TurnResult, error) {
	t.calls++
	t.req = r
	return c.TurnResult{Status: c.TurnCompleted}, nil
}
func (t *productTurns) RetryFailed(_ context.Context, u, client c.ID) error {
	t.retryCalls++
	t.user, t.client = u, client
	return nil
}
func (t *productTurns) RecoverNext(_ context.Context, u c.ID) (c.TurnResult, error) {
	t.recoverCalls++
	t.user = u
	return c.TurnResult{Status: c.TurnCompleted}, nil
}

type productMem struct{ Memory }

func (productMem) Reference() c.BundleRef { return c.BundleRef{Version: "synthetic"} }

type accounts struct{ Accounts }

func TestOwnershipOnboardingAndInputPrecedeAdmission(t *testing.T) {
	repo := &productRepo{}
	turns := &productTurns{}
	svc, e := New(turns, productMem{}, repo, accounts{}, Options{ConsentKeys: []string{"reviewed_chat"}, ChatConsentKey: "reviewed_chat"})
	if e != nil {
		t.Fatal(e)
	}
	req := c.TurnRequest{UserID: c.NewID(), ConversationID: c.NewID(), ClientTurnID: c.NewID(), Origin: c.OriginUser, Text: "cafe\u0301", Locale: "en", TZ: "UTC"}
	repo.ownerErr = httpx.ErrNotFound
	if _, e = svc.Process(context.Background(), req); !errors.Is(e, httpx.ErrNotFound) {
		t.Fatal("ownership bypass", e)
	}
	repo.ownerErr = nil
	repo.readyErr = httpx.ErrOnboardingRequired
	if _, e = svc.Process(context.Background(), req); !errors.Is(e, httpx.ErrOnboardingRequired) {
		t.Fatal("onboarding bypass")
	}
	repo.readyErr = nil
	for _, bad := range []func(*c.TurnRequest){func(r *c.TurnRequest) { r.TZ = "invalid-zone" }, func(r *c.TurnRequest) { r.Locale = "not_a_locale!" }, func(r *c.TurnRequest) { r.Text = "\x00" }, func(r *c.TurnRequest) { r.Origin = c.OriginWakeUp }, func(r *c.TurnRequest) { r.ClientTurnID = c.ID{} }} {
		v := req
		bad(&v)
		if _, e = svc.Process(context.Background(), v); !errors.Is(e, httpx.ErrInvalid) {
			t.Fatal("invalid reached core", e)
		}
	}
	if turns.calls != 0 {
		t.Fatal("invalid admitted")
	}
	if _, e = svc.Process(context.Background(), req); e != nil || turns.req.Text != "café" || turns.req.Channel != "app_chat" {
		t.Fatal("ingress normalization", e)
	}
}

func TestChatProcessRetryAndRecoveryNeverReadConsent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		consent map[string]bool
		readErr error
	}{
		{name: "missing"},
		{name: "empty", consent: map[string]bool{}},
		{name: "false", consent: map[string]bool{"reviewed_chat": false}},
		{name: "true", consent: map[string]bool{"reviewed_chat": true}},
		{name: "unavailable_controls", readErr: errors.New("synthetic controls failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &productRepo{controls: Controls{Consent: tc.consent}, controlsErr: tc.readErr}
			turns := &productTurns{}
			svc, err := New(turns, productMem{}, repo, accounts{}, Options{ConsentKeys: []string{"reviewed_chat"}, ChatConsentKey: "reviewed_chat"})
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			req := c.TurnRequest{UserID: c.NewID(), ConversationID: c.NewID(), ClientTurnID: c.NewID(), Origin: c.OriginUser, Text: "synthetic message"}
			if result, err := svc.Process(ctx, req); err != nil || result.Status != c.TurnCompleted || turns.calls != 1 {
				t.Fatalf("chat was blocked: %+v %v", result, err)
			}
			if result, err := svc.Retry(ctx, req.UserID, req.ConversationID, req.ClientTurnID); err != nil || result.Status != c.TurnQueued || turns.retryCalls != 1 || turns.user != req.UserID || turns.client != req.ClientTurnID {
				t.Fatalf("retry was blocked: %+v %v", result, err)
			}
			if result, err := svc.Recover(ctx, req.UserID); err != nil || result.Status != c.TurnCompleted || turns.recoverCalls != 1 || turns.user != req.UserID {
				t.Fatalf("recovery was blocked: %+v %v", result, err)
			}
			if repo.controlsCalls != 0 {
				t.Fatal("chat read consent controls")
			}
		})
	}
}

func TestLegacyChatConsentOptionIsIgnored(t *testing.T) {
	if _, err := New(&productTurns{}, productMem{}, &productRepo{}, accounts{}, Options{ChatConsentKey: "obsolete_chat_key"}); err != nil {
		t.Fatal("legacy chat key required a consent mapping", err)
	}
}

func TestRetryStillRequiresOwnershipAndOnboarding(t *testing.T) {
	for _, gate := range []string{"owner", "onboarding"} {
		t.Run(gate, func(t *testing.T) {
			repo := &productRepo{}
			want := httpx.ErrNotFound
			if gate == "owner" {
				repo.ownerErr = want
			} else {
				want = httpx.ErrOnboardingRequired
				repo.readyErr = want
			}
			turns := &productTurns{}
			svc, err := New(turns, productMem{}, repo, accounts{}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Retry(context.Background(), c.NewID(), c.NewID(), c.NewID()); !errors.Is(err, want) || turns.retryCalls != 0 || repo.turnReads != 0 {
				t.Fatal("retry bypassed authorization", err)
			}
		})
	}
}
func TestControlVocabularyAndMetadataAreExplicit(t *testing.T) {
	repo := &productRepo{}
	turns := &productTurns{}
	s, e := New(turns, productMem{}, repo, accounts{}, Options{ConsentKeys: []string{"reviewed_chat"}})
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	u := c.NewID()
	if e = s.SetControls(ctx, u, c.NewID(), nil, map[string]bool{"grant_all": true}, nil); !errors.Is(e, httpx.ErrInvalid) {
		t.Fatal("invented consent accepted")
	}
	if e = s.SetControls(ctx, u, c.ID{}, nil, map[string]bool{"reviewed_chat": true}, nil); !errors.Is(e, httpx.ErrInvalid) {
		t.Fatal("missing operation accepted")
	}
	keys := s.Keys()
	keys[0] = "mutated"
	if s.Keys()[0] != "reviewed_chat" {
		t.Fatal("mutable consent keys")
	}
}
