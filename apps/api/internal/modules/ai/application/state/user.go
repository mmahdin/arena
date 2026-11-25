package state

import (
	"math"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func riskRank(r c.RiskLevel) int {
	switch r {
	case c.RiskNone:
		return 0
	case c.RiskLow:
		return 1
	case c.RiskElevated:
		return 2
	case c.RiskHigh:
		return 3
	case c.RiskImminent:
		return 4
	default:
		return -1
	}
}
func (s *Service) validateUser(snap contextstate.Snapshot, pol contextstate.Policy, d *c.StateDelta, reject rejectFunc) {
	patch := &d.User
	if patch.Risk != nil {
		tr := patch.Risk
		old := snap.User.Risk.Level
		if riskRank(old) < 0 {
			old = c.RiskElevated
		}
		if tr.TurnID != d.TurnID || !tr.To.Valid() || tr.At.IsZero() {
			patch.Risk = nil
			reject("user.risk", "ST-010", "invalid risk transition")
		} else if riskRank(tr.To) < riskRank(old) {
			// No candidate lowering meets the reviewed safety policy without
			// authoritative user-statement evidence (not in StateDelta).
			patch.Risk = nil
			reject("user.risk", "ST-010", "risk de-escalation requires separate validated safety evidence")
		} else if riskRank(tr.To) == riskRank(old) {
			patch.Risk = nil
			reject("user.risk", "ST-010", "no risk change")
		} else if tr.From != old {
			tr.From = old
			reject("user.risk.from", "ST-011", "raised risk from actual persisted state")
		}
	}
	for key, value := range patch.EngagementDelta {
		if key == "" || len(key) > 64 || math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e6 {
			delete(patch.EngagementDelta, key)
			reject("user.engagement_delta", "ST-011", "unbounded engagement change")
		}
	}
	if p := patch.PrefsPatch; p != nil {
		if p.Length != nil && !p.Length.Valid() {
			p.Length = nil
			reject("user.prefs.length", "ST-011", "invalid response length")
		}
		if p.Tone != nil && !p.Tone.Valid() {
			p.Tone = nil
			reject("user.prefs.tone", "ST-011", "invalid tone")
		}
		if p.AddressForm != nil && len([]rune(*p.AddressForm)) > 60 {
			p.AddressForm = nil
			reject("user.prefs.address_form", "ST-011", "address form too long")
		}
		sanitize := func(in []string) []string {
			out := []string{}
			seen := map[string]bool{}
			for _, v := range in {
				v = strings.TrimSpace(v)
				if v != "" && len([]rune(v)) <= 80 && !seen[v] && len(out) < 20 {
					out = append(out, v)
					seen[v] = true
				}
			}
			return out
		}
		oldAdded, oldRemoved := len(p.AvoidAdd), len(p.AvoidRemove)
		p.AvoidAdd = sanitize(p.AvoidAdd)
		p.AvoidRemove = sanitize(p.AvoidRemove)
		if len(p.AvoidAdd) != oldAdded || len(p.AvoidRemove) != oldRemoved {
			reject("user.prefs.avoid", "ST-011", "duplicate or invalid preference item")
		}
	}
	_ = pol
}

func (s *Service) validateOperations(snap contextstate.Snapshot, d *c.StateDelta, reject rejectFunc) {
	known := map[c.ID]c.OpenLoop{}
	key := func(kind string, ref *c.ID) string {
		if ref == nil {
			return ""
		}
		return kind + ":" + ref.String()
	}
	keys := map[string]bool{}
	for _, v := range snap.OpenLoops {
		known[v.ID] = v
		if k := key(v.Kind, v.Ref); k != "" {
			keys[k] = true
		}
	}
	loops := make([]c.OpenLoopOp, 0, len(d.OpenLoops))
	for i, op := range d.OpenLoops {
		path := "open_loops[" + itoa(i) + "]"
		switch op.Op {
		case "add":
			if !c.ValidID(op.Loop.ID) || strings.TrimSpace(op.Loop.Kind) == "" || strings.TrimSpace(op.Loop.Text) == "" || len([]rune(op.Loop.Text)) > 300 {
				reject(path, "ST-040", "invalid loop")
				continue
			}
			if _, exists := known[op.Loop.ID]; exists {
				reject(path, "ST-040", "loop ID already exists")
				continue
			}
			if k := key(op.Loop.Kind, op.Loop.Ref); k != "" {
				if keys[k] {
					reject(path, "ST-040", "duplicate open loop merged by keeping existing")
					continue
				}
				keys[k] = true
			}
			known[op.Loop.ID] = op.Loop
			loops = append(loops, op)
		case "close", "expire":
			if _, ok := known[op.Loop.ID]; !ok {
				reject(path, "ST-040", "loop not open for this user")
				continue
			}
			delete(known, op.Loop.ID)
			loops = append(loops, op)
		default:
			reject(path, "ST-040", "unknown loop operation")
		}
	}
	d.OpenLoops = loops
	now := s.now().UTC()
	scheduled := make([]time.Time, 0, len(snap.FollowUps)+len(d.FollowUps))
	for _, f := range snap.FollowUps {
		if f.Status == "scheduled" {
			scheduled = append(scheduled, f.DueAt.UTC())
		}
	}
	follows := make([]c.FollowUpOp, 0, len(d.FollowUps))
	loc, err := time.LoadLocation(snap.User.TZ)
	if err != nil {
		loc = time.UTC
	}
	for i, op := range d.FollowUps {
		path := "follow_ups[" + itoa(i) + "]"
		switch op.Op {
		case "schedule":
			if !snap.User.ProactiveOptIn {
				reject(path, "ST-050", "user did not opt into proactive messages")
				continue
			}
			if !c.ValidID(op.FollowUpID) || op.DueAt.IsZero() || !op.DueAt.After(now) {
				reject(path, "ST-050", "invalid or past due time")
				continue
			}
			if quietAt(op.DueAt.In(loc), snap.User.QuietHours) {
				reject(path, "ST-050", "time falls inside quiet hours")
				continue
			}
			blocked := false
			for _, at := range scheduled {
				d := at.Sub(op.DueAt.UTC())
				if d < 0 {
					d = -d
				}
				if d < 24*time.Hour {
					blocked = true
					break
				}
			}
			if blocked {
				reject(path, "ST-050", "one follow-up per 24 hours")
				continue
			}
			scheduled = append(scheduled, op.DueAt.UTC())
			follows = append(follows, op)
		case "cancel":
			known := false
			for _, f := range snap.FollowUps {
				if f.ID == op.FollowUpID && f.UserID == d.UserID && f.Status == "scheduled" {
					known = true
					break
				}
			}
			if known {
				follows = append(follows, op)
			} else {
				reject(path, "ST-050", "follow-up not scheduled for owner")
			}
		default:
			reject(path, "ST-050", "unknown follow-up operation")
		}
	}
	d.FollowUps = follows
	events := make([]c.DomainEvent, 0, len(d.Events))
	for i, event := range d.Events {
		if !event.Type.Valid() {
			reject("events["+itoa(i)+"]", "ST-001", "unknown event type")
			continue
		}
		events = append(events, event)
	}
	d.Events = events
}

func quietAt(when time.Time, hours *c.QuietHours) bool {
	if hours == nil {
		return false
	}
	start, e1 := time.Parse("15:04", hours.Start)
	end, e2 := time.Parse("15:04", hours.End)
	if e1 != nil || e2 != nil {
		return true
	} // unsafe quiet-hours config is fail-closed
	lo := start.Hour()*60 + start.Minute()
	hi := end.Hour()*60 + end.Minute()
	at := when.Hour()*60 + when.Minute()
	if lo == hi {
		return true
	}
	if lo < hi {
		return at >= lo && at < hi
	}
	return at >= lo || at < hi
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
