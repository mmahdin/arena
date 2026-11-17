package llm

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type DailyTotals struct {
	Calls, Attempts     int
	InTokens, OutTokens int64
	CachedTokens        int64
	CostMicroUSD        int64
}

type meterKey struct {
	User contract.ID
	Day  string
}

// MemoryLedger is a privacy-minimal, concurrency-safe fake/in-process meter
// for WP-04 and development. No entries are purged/evicted; it never mutates
// conversation/state data. Production quotas require a durable SQL ledger wired
// at WP-05 or later: process-local pre-checks cannot reserve in-flight spend.
type MemoryLedger struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[meterKey]DailyTotals
}

func NewMemoryLedger(now func() time.Time) *MemoryLedger {
	if now == nil {
		now = time.Now
	}
	return &MemoryLedger{now: now, items: make(map[meterKey]DailyTotals)}
}
func (m *MemoryLedger) key(user contract.ID) meterKey {
	return meterKey{User: user, Day: m.now().UTC().Format("2006-01-02")}
}
func (m *MemoryLedger) Check(ctx context.Context, user contract.ID, quota DailyQuota) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !contract.ValidID(user) {
		return ErrConfig
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.items[m.key(user)]
	if quota.Calls > 0 && t.Calls >= quota.Calls || quota.InTokens > 0 && t.InTokens >= quota.InTokens ||
		quota.OutTokens > 0 && t.OutTokens >= quota.OutTokens || quota.CostMicroUSD > 0 && t.CostMicroUSD >= quota.CostMicroUSD {
		return ErrQuota
	}
	return nil
}
func (m *MemoryLedger) Record(_ context.Context, user contract.ID, usage contract.LLMUsage) error {
	if !contract.ValidID(user) || usage.InTokens < 0 || usage.OutTokens < 0 || usage.CachedTokens < 0 || usage.CachedTokens > usage.InTokens || usage.CostMicroUSD < 0 || usage.Attempts < 0 {
		return ErrUsage
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.key(user)
	t := m.items[key]
	if t.Calls == math.MaxInt || int64(usage.InTokens) > math.MaxInt64-t.InTokens || int64(usage.OutTokens) > math.MaxInt64-t.OutTokens || int64(usage.CachedTokens) > math.MaxInt64-t.CachedTokens || usage.CostMicroUSD > math.MaxInt64-t.CostMicroUSD || usage.Attempts > math.MaxInt-t.Attempts {
		return ErrUsage
	}
	t.Calls++
	t.Attempts += usage.Attempts
	t.InTokens += int64(usage.InTokens)
	t.OutTokens += int64(usage.OutTokens)
	t.CachedTokens += int64(usage.CachedTokens)
	t.CostMicroUSD += usage.CostMicroUSD
	m.items[key] = t
	return nil
}
func (m *MemoryLedger) Snapshot(user contract.ID, day time.Time) DailyTotals {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.items[meterKey{User: user, Day: day.UTC().Format("2006-01-02")}]
}
