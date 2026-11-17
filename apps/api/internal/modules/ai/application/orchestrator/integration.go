package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/assembler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

// IntegratedOptions makes WP-06 and WP-08 dependencies explicit. The bundle
// must have been release-verified and loaded through safety.Load; it must be
// the same immutable version returned by Bundles for each admitted turn.
// Classifier is typically safety.NewLLMClassifier backed by the WP-04 client.
// There is deliberately no production HTTP/composition switch in this WP.
type IntegratedOptions struct {
	Reader     contextstate.Reader
	Bundles    ports.BundleRegistry
	Assets     *safety.Assets
	Classifier safety.Classifier
	SecondLook safety.SecondLook
	// ONLY for isolated, non-clinical integration fixtures. Never set in a
	// production composition root; synthetic WP-03 assets are not approved.
	AllowSyntheticForTests bool
	Now                    func() time.Time
}

type integration struct {
	reader                 contextstate.Reader
	registry               ports.BundleRegistry
	assets                 *safety.Assets
	validator              ports.Validator
	fsm                    *safety.FSM
	committer              *state.Service
	now                    func() time.Time
	allowSyntheticForTests bool
}

// NewIntegrated opts IN to the complete WP-06/WP-08 deterministic pathway.
// Unlike the original WP-05 New, no caller may substitute a permissive Safety
// Gate or a state-less assembler: they are constructed from the actual modules.
// The one WP-02 TurnFinalizer is wrapped by WP-08's validating finalizer, not
// replaced by another database writer. Brain/Generator/Validator still need
// real reviewed implementations before production use (later work packages).
func NewIntegrated(d Dependencies, opts Options, i IntegratedOptions) (*Service, error) {
	if i.Reader == nil || i.Bundles == nil || i.Assets == nil || i.Classifier == nil || d.Store == nil || d.Finalizer == nil {
		return nil, errors.New("integrated orchestrator requires WP-02 store/finalizer, WP-03 bundle, WP-06 assets/classifier and WP-08 reader")
	}
	now := i.Now
	if now == nil {
		now = time.Now
	}
	gate, err := safety.NewGate(i.Assets, i.Classifier, i.SecondLook)
	if err != nil {
		return nil, err
	}
	planner, err := safety.NewPlanner(i.Assets, now)
	if err != nil {
		return nil, err
	}
	fsm, err := safety.NewFSM(i.Assets)
	if err != nil {
		return nil, err
	}
	// Only this registry can serve the integrated assembler/committer: after
	// admission ProcessTurn pins a single handle in the request context. This
	// eliminates silent mid-turn Knowledge Bundle switches on multiple reads.
	pinned := pinnedRegistry{}
	assemblerService, err := assembler.New(i.Reader, d.Store, pinned)
	if err != nil {
		return nil, err
	}
	assemblerService.WithClock(now)
	committer, err := state.New(i.Reader, pinned)
	if err != nil {
		return nil, err
	}
	committer.WithClock(now)
	validated, err := state.NewValidatedFinalizer(d.Finalizer, committer)
	if err != nil {
		return nil, err
	}
	d.Safety = gate
	d.Crisis = planner
	d.Assembler = assemblerService
	d.Finalizer = validated
	// The integrated path must never admit an arbitrary caller-provided
	// fallback. All non-crisis fallback text comes from this pinned WP-03
	// bundle; crisis fallback comes from the verified CrisisPlan itself.
	opts.Fallback = i.Assets.FallbackFunc
	o, err := New(d, opts)
	if err != nil {
		return nil, err
	}
	o.integration = &integration{reader: i.Reader, registry: i.Bundles, assets: i.Assets,
		fsm: fsm, committer: committer, validator: d.Validator, now: now, allowSyntheticForTests: i.AllowSyntheticForTests}
	return o, nil
}

type pinnedBundleKey struct{}
type pinnedBundle struct {
	user  contract.ID
	value ports.Bundle
}

type pinnedRegistry struct{}

var _ ports.BundleRegistry = pinnedRegistry{}

func (pinnedRegistry) Active(ctx context.Context, user contract.ID) (ports.Bundle, error) {
	pinned, ok := ctx.Value(pinnedBundleKey{}).(pinnedBundle)
	if !ok || pinned.user != user || pinned.value == nil {
		return nil, errors.New("Knowledge Bundle not pinned to this authenticated turn")
	}
	return pinned.value, nil
}

func (i *integration) pin(ctx context.Context, a contract.Admission) (context.Context, error) {
	b, err := i.registry.Active(ctx, a.UserID)
	if err != nil || b == nil {
		return ctx, coreError(contract.ErrBundle, "bundle", err)
	}
	ref := b.Reference()
	approved := false
	if v, ok := b.(interface{ SyntheticOnly() bool }); ok {
		approved = !v.SyntheticOnly() || i.allowSyntheticForTests
	}
	if !approved || ref.Version == "" || ref.Version != i.assets.Reference().Version ||
		!reflect.DeepEqual(ref.PromptSHAs, i.assets.Reference().PromptSHAs) {
		return ctx, coreError(contract.ErrBundle, "bundle", fmt.Errorf("unapproved or mismatched pinned bundle"))
	}
	// A production WP-07 implementation must be pinned to these same
	// immutable bytes. Synthetic test validators are only allowed behind the
	// explicit test-only flag, never through the production composition root.
	if !i.allowSyntheticForTests {
		v, ok := i.validator.(interface{ Reference() contract.BundleRef })
		if !ok || v.Reference().Version != ref.Version || !reflect.DeepEqual(v.Reference().PromptSHAs, ref.PromptSHAs) {
			return ctx, coreError(contract.ErrBundle, "validator_bundle", fmt.Errorf("validator bundle differs from safety bundle"))
		}
	}
	ctx = context.WithValue(ctx, pinnedBundleKey{}, pinnedBundle{user: a.UserID, value: b})
	ctx = safety.WithCallMeta(ctx, contract.CallMeta{UserID: a.UserID, TurnID: a.TurnID, Bundle: ref})
	ctx = validator.WithCallMeta(ctx, contract.CallMeta{UserID: a.UserID, TurnID: a.TurnID, Bundle: ref})
	return ctx, nil
}

func (o *Service) bundleReference(ctx context.Context) contract.BundleRef {
	if o.integration == nil {
		return contract.BundleRef{}
	}
	pinned, ok := ctx.Value(pinnedBundleKey{}).(pinnedBundle)
	if !ok || pinned.value == nil {
		return contract.BundleRef{}
	}
	return pinned.value.Reference()
}
