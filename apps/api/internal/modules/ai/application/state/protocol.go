package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (s *Service) validateProtocol(ctx context.Context, snap contextstate.Snapshot, pol contextstate.Policy, d *c.StateDelta, reject rejectFunc) error {
	p := d.Protocol
	if p != nil {
		existing := snap.Protocol
		if !c.ValidID(p.InstanceID) || p.Create == nil && (existing == nil || existing.ID != p.InstanceID || existing.UserID != d.UserID) ||
			p.Create != nil && existing != nil && existing.ID == p.InstanceID {
			d.Protocol = nil
			reject("protocol", "ST-022", "instance is not owned or new ID is already in use")
		} else if p.Create != nil && d.ExpectedVersions.Protocol != 0 || p.Create == nil && d.ExpectedVersions.Protocol != existing.Version {
			return stateErr(c.ErrStateConflict, fmt.Errorf("ST-001: protocol version changed"))
		} else {
			if p.AttemptsInc < 0 {
				p.AttemptsInc = 0
				reject("protocol.attempts_inc", "ST-021", "negative attempt delta")
			}
			if p.StuckInc < 0 {
				p.StuckInc = 0
				reject("protocol.stuck_inc", "ST-021", "negative stuck delta")
			}
			if p.Create != nil {
				if p.Create.ProtocolID == "" || p.Create.Version == "" || p.Create.OfferedAt.IsZero() {
					d.Protocol = nil
					reject("protocol.create", "ST-021", "incomplete offer")
				} else if p.Create.ThreadID != nil {
					if _, ok := contextstate.FindThread(snap, *p.Create.ThreadID); !ok && (d.Thread == nil || d.Thread.Create == nil || d.Thread.ThreadID != *p.Create.ThreadID) {
						d.Protocol = nil
						reject("protocol.create.thread_id", "ST-022", "unowned thread")
					}
				}
				if existing != nil && (existing.Status == c.PIOffered || existing.Status == c.PIActive) {
					d.Protocol = nil
					reject("protocol.create", "ST-022", "only one live protocol allowed")
				}
			}
		}
	}
	p = d.Protocol
	if p != nil {
		id := string("")
		version := ""
		oldStep := ""
		if p.Create != nil {
			id = p.Create.ProtocolID
			version = p.Create.Version
		} else {
			id = snap.Protocol.ProtocolID
			version = snap.Protocol.ProtocolVersion
			oldStep = snap.Protocol.StepID
		}
		graph, err := contextstate.ProtocolGraph(ctx, pol.Handle, id)
		if err != nil || graph.Version != version {
			// Never authorize a protocol state change without its exact
			// versioned step graph.
			d.Protocol = nil
			reject("protocol", "ST-021", "pinned protocol graph unavailable or version mismatch")
		} else {
			if p.NewStatus != nil && !p.NewStatus.Valid() {
				p.NewStatus = nil
				reject("protocol.new_status", "ST-021", "invalid status")
			}
			if p.NewStatus != nil {
				from := c.PIOffered
				if p.Create == nil {
					from = snap.Protocol.Status
				}
				if !validProtocolStatusChange(from, *p.NewStatus) {
					p.NewStatus = nil
					reject("protocol.new_status", "ST-021", "disallowed protocol status transition")
				}
			}
			if p.NewStatus != nil && *p.NewStatus == c.PIActive {
				consentAt := p.ConsentAt
				consentID := p.ConsentMessageID
				if snap.Protocol != nil && p.Create == nil {
					if consentAt == nil {
						consentAt = snap.Protocol.ConsentAt
					}
					if consentID == nil {
						consentID = snap.Protocol.ConsentMessageID
					}
				}
				valid := consentAt != nil && !consentAt.IsZero() && consentID != nil && c.ValidID(*consentID)
				if valid {
					roles, err := s.reader.UserEvidence(ctx, d.UserID, []c.ID{*consentID})
					if err != nil {
						return err
					}
					valid = roles[*consentID]
				}
				if !valid {
					p.NewStatus = nil
					reject("protocol.new_status", "ST-020", "active requires verified user consent message and timestamp")
				}
			}
			if p.NewStep != nil {
				valid := graphHasStep(graph, *p.NewStep)
				if valid && oldStep != "" && *p.NewStep != oldStep {
					valid = reachableStep(graph, oldStep, *p.NewStep)
				}
				if !valid {
					p.NewStep = nil
					reject("protocol.new_step", "ST-021", "step not in the permitted graph transition")
				}
			}
			for key, value := range p.Inputs {
				if strings.TrimSpace(key) == "" || len(key) > 100 || wordCount(value) > 30 {
					delete(p.Inputs, key)
					reject("protocol.inputs", "ST-021", "unexpected or unbounded protocol input")
				}
			}
			// WP-13 owns structured protocol input capture and outcomes.
			// This validates graph shape only, never advances by itself.
		}
	}
	effectiveRisk := snap.User.Risk.Level
	if d.User.Risk != nil && riskRank(d.User.Risk.To) > riskRank(effectiveRisk) {
		effectiveRisk = d.User.Risk.To
	}
	if d.SchemaVersion == "state_delta.v2" && riskRank(effectiveRisk) >= 2 {
		if snap.Protocol != nil && snap.Protocol.Status == c.PIActive && (d.Protocol == nil || d.Protocol.NewStatus == nil || (*d.Protocol.NewStatus != c.PIPaused && *d.Protocol.NewStatus != c.PIStopped)) {
			reject("protocol", "ST-012", "active protocol needs a resolved interruption")
		}
		return nil
	}
	if riskRank(effectiveRisk) >= 2 {
		if d.Protocol != nil && d.Protocol.NewStatus != nil && *d.Protocol.NewStatus == c.PIActive {
			d.Protocol.NewStatus = nil
			reject("protocol.new_status", "ST-012", "elevated risk blocks activation")
		}
		if snap.Protocol != nil && snap.Protocol.Status == c.PIActive {
			// Raise/degradation pauses an existing protocol in the same UoW.
			if d.Protocol == nil {
				d.Protocol = &c.ProtocolPatch{InstanceID: snap.Protocol.ID, Inputs: map[string]string{}, Reason: "too_distressed"}
				d.ExpectedVersions.Protocol = snap.Protocol.Version
			}
			if d.Protocol.InstanceID == snap.Protocol.ID {
				paused := c.PIPaused
				d.Protocol.NewStatus = &paused
				d.Protocol.NewStep = nil
				reject("protocol.new_status", "ST-012", "paused active protocol while risk elevated")
			}
		}
	}
	return nil
}

func graphHasStep(graph bundle.Protocol, id string) bool {
	for _, step := range graph.Steps {
		if step.ID == id {
			return true
		}
	}
	return false
}
func reachableStep(graph bundle.Protocol, from, to string) bool {
	for _, v := range graph.Steps {
		if v.ID != from {
			continue
		}
		if v.OnSuccess == to || v.OnStuck.Then == to || v.OnDistress == to || v.OnDecline == to {
			return true
		}
	}
	return false
}

// Lifecycle guards are kept separate from the clinical step graph. Protocol
// content and intervention eligibility remain owned by WP-13.
func validProtocolStatusChange(from, to c.ProtocolStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case c.PIOffered:
		return to == c.PIActive || to == c.PIDeclined || to == c.PIStopped
	case c.PIActive:
		return to == c.PIPaused || to == c.PIStopped || to == c.PICompleted
	case c.PIPaused:
		return to == c.PIActive || to == c.PIStopped
	default:
		return false
	}
}
