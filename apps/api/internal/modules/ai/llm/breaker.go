package llm

import (
	"sync"
	"time"
)

type breakerEvent struct {
	at     time.Time
	failed bool
}

type circuit struct {
	until   time.Time
	probing bool
	events  []breakerEvent
}

// breakers are scoped to provider+model, never a whole tier. A half-open
// circuit admits exactly one probe; other callers go to the next model.
type breakers struct {
	mu     sync.Mutex
	config Config
	models map[string]*circuit
}

func newBreakers(config Config) *breakers {
	return &breakers{config: config, models: map[string]*circuit{}}
}
func (b *breakers) get(key string) *circuit {
	s := b.models[key]
	if s == nil {
		s = &circuit{}
		b.models[key] = s
	}
	return s
}
func (b *breakers) allow(key string, now time.Time) (bool, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.get(key)
	if !s.until.IsZero() {
		if now.Before(s.until) || s.probing {
			return false, false
		}
		s.probing = true
		return true, true
	}
	return true, false
}
func (b *breakers) record(key string, now time.Time, probe bool, failed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.get(key)
	if probe {
		s.probing = false
		s.events = nil
		if failed {
			s.until = now.Add(b.config.BreakerCooldown)
		} else {
			s.until = time.Time{}
		}
		return
	}
	if !s.until.IsZero() {
		return
	}
	window := now.Add(-b.config.BreakerWindow)
	kept := s.events[:0]
	for _, event := range s.events {
		if !event.at.Before(window) {
			kept = append(kept, event)
		}
	}
	s.events = append(kept, breakerEvent{at: now, failed: failed})
	if len(s.events) < b.config.BreakerMinSamples {
		return
	}
	var failures int
	for _, event := range s.events {
		if event.failed {
			failures++
		}
	}
	if float64(failures)/float64(len(s.events)) >= b.config.BreakerErrorRate {
		s.until = now.Add(b.config.BreakerCooldown)
		s.events = nil
	}
}
