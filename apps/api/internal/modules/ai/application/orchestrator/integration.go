package orchestrator

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"reflect"
	"time"
)

type integration struct {
	unified                ports.UnifiedModel
	selector               KnowledgeSelector
	persistent             contextstate.PersistentReader
	unifiedMemory          UnifiedMemory
	unifiedReflection      UnifiedReflection
	reader                 contextstate.Reader
	registry               ports.BundleRegistry
	protocol               ports.ProtocolEngine
	committer              *state.Service
	now                    func() time.Time
	allowSyntheticForTests bool
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
	if !reflect.DeepEqual(ref, i.selector.Reference()) {
		return ctx, coreError(contract.ErrBundle, "jev_bundle", nil)
	}
	{
		source, ok := b.(interface{ SyntheticOnly() bool })
		if !ok || source.SyntheticOnly() && !i.allowSyntheticForTests || !reflect.DeepEqual(ref, i.unified.Reference()) {
			return ctx, coreError(contract.ErrBundle, "unified_bundle", nil)
		}
		if i.unifiedReflection != nil && !reflect.DeepEqual(i.unifiedReflection.Reference(), ref) {
			return ctx, coreError(contract.ErrBundle, "reflection_bundle", nil)
		}
		if i.unifiedMemory != nil && !reflect.DeepEqual(i.unifiedMemory.Reference(), ref) {
			return ctx, coreError(contract.ErrBundle, "memory_bundle", nil)
		}
		if i.protocol != nil {
			if !reflect.DeepEqual(i.protocol.Reference(), ref) {
				return ctx, coreError(contract.ErrBundle, "protocol_bundle", nil)
			}
		}
		ctx = context.WithValue(ctx, pinnedBundleKey{}, pinnedBundle{user: a.UserID, value: b})
		return retrieval.PinBundle(ctx, a.UserID, b), nil
	}

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
