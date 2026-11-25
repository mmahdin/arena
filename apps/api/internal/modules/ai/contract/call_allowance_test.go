package contract_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestTurnCallAllowanceSharedAcrossDerivedContexts(t *testing.T) {
	ctx := c.WithCallAllowance(context.Background(), 3)
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for n := 0; n < 30; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			child, cancel := context.WithCancel(ctx)
			defer cancel()
			if c.ReserveLLMCall(child) {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 3 || c.ReserveLLMCall(ctx) {
		t.Fatal("turn allowance was not shared/atomic", accepted.Load())
	}
	if !c.ReserveLLMCall(c.WithCallAllowance(context.Background(), 1)) {
		t.Fatal("allowance leaked into another turn")
	}
}
