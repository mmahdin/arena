// Package protocol implements WP-13's deterministic, version-pinned intervention
// engine. All intervention wording comes from release-verified Knowledge Bundles.
package protocol

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

var ErrUnreviewed = errors.New("protocol: unreviewed or unapproved intervention")
var ErrInvalid = errors.New("protocol: invalid action or unavailable pinned definition")
var ErrConsent = errors.New("protocol: explicit authenticated user consent required")

type Assets struct {
	ref       c.BundleRef
	synthetic bool
	protocols map[string]bundle.Protocol
	offers    bundle.ProtocolOfferTexts
}

type LoadOptions struct{ AllowSyntheticForTests bool }

// Load rejects unreviewed or synthetic content by default. In production the
// registry must independently verify the signed bundle release; matching
// clinical approval hashes are a second, protocol-specific gate. This test
// override never enables a production default.
func Load(ctx context.Context, source ports.Bundle, opts LoadOptions) (*Assets, error) {
	if source == nil {
		return nil, ErrUnreviewed
	}
	names, ok := source.(interface{ ArtifactNames() []string })
	if !ok {
		return nil, ErrUnreviewed
	}
	synthetic, ok := source.(interface{ SyntheticOnly() bool })
	if !ok || synthetic.SyntheticOnly() && !opts.AllowSyntheticForTests {
		return nil, ErrUnreviewed
	}
	ref := source.Reference()
	if ref.Version == "" {
		return nil, ErrInvalid
	}
	var manifest bundle.Manifest
	if err := decode(ctx, source, "bundle.yaml", &manifest); err != nil || "kb@"+manifest.Version != ref.Version || manifest.SyntheticOnly != synthetic.SyntheticOnly() {
		return nil, ErrUnreviewed
	}
	out := &Assets{ref: ref, synthetic: synthetic.SyntheticOnly(), protocols: map[string]bundle.Protocol{}}
	all := append([]string(nil), names.ArtifactNames()...)
	sort.Strings(all)
	for _, path := range all {
		if !strings.HasPrefix(path, "protocols/") || !strings.HasSuffix(path, ".yaml") {
			continue
		}
		raw, err := source.Artifact(ctx, path)
		if err != nil {
			return nil, ErrInvalid
		}
		var p bundle.Protocol
		if err := yamlDecode(raw, &p); err != nil || p.Schema != "protocol.v1" || p.ID == "" || path != "protocols/"+p.ID+".yaml" || p.Version == "" || len(p.Steps) == 0 {
			return nil, ErrInvalid
		}
		if _, exists := out.protocols[p.ID]; exists {
			return nil, ErrInvalid
		}
		if !out.synthetic {
			digest := sha256.Sum256(raw)
			sha := hex.EncodeToString(digest[:])
			approved := false
			for _, a := range manifest.ClinicalReview.Approvals {
				if a.Artifact == path && a.SHA256 == sha && a.ReviewerID != "" && a.Date != "" {
					approved = true
					break
				}
			}
			if !approved || manifest.Checksums.Artifacts[path] != sha {
				return nil, fmt.Errorf("%w: %s", ErrUnreviewed, path)
			}
		}
		if !validGraph(p) {
			return nil, fmt.Errorf("%w: %s", ErrInvalid, path)
		}
		out.protocols[p.ID] = p
	}
	if len(out.protocols) > 0 {
		if err := decode(ctx, source, "templates/protocol_offer.yaml", &out.offers); err != nil || out.offers.Schema != "protocol_offer.v1" {
			return nil, ErrInvalid
		}
	}
	for _, p := range out.protocols {
		for _, id := range p.Offer.WhyTemplates {
			if strings.TrimSpace(out.offers.WhyTemplates[id].Text) == "" {
				return nil, ErrInvalid
			}
		}
		for _, id := range p.Offer.MustSay {
			if strings.TrimSpace(out.offers.MustSay[id].Text) == "" {
				return nil, ErrInvalid
			}
		}
		if !out.synthetic && (len(p.Offer.WhyTemplates) == 0 || len(p.Offer.MustSay) == 0) {
			return nil, ErrUnreviewed
		}
	}
	return out, nil
}
func decode(ctx context.Context, source ports.Bundle, p string, dest any) error {
	raw, e := source.Artifact(ctx, p)
	if e != nil {
		return e
	}
	return yamlDecode(raw, dest)
}
func yamlDecode(raw []byte, dest any) error {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(dest); err != nil {
		return err
	}
	var next any
	if err := dec.Decode(&next); err != io.EOF {
		return ErrInvalid
	}
	return nil
}
func validGraph(p bundle.Protocol) bool {
	if !c.RiskLevel(p.Eligibility.RequireRiskAtMost).Valid() || readinessRank(Readiness(p.Eligibility.MinUserReadiness)) < 0 || len(p.Eligibility.RequirePhase) == 0 || p.Eligibility.MaxRunsPerThread < 1 || p.Eligibility.DeclineCooldownDays < 0 || p.Eligibility.MinDaysBetweenRuns < 0 || p.EstMinutes < 1 || p.Name == "" || p.OneLiner == "" {
		return false
	}
	ids := map[string]bool{}
	for _, phase := range p.Eligibility.RequirePhase {
		if !c.ThreadPhase(phase).Valid() {
			return false
		}
	}
	for _, s := range p.Steps {
		if s.ID == "" || ids[s.ID] || s.Instruction == "" || s.Goal == "" || s.SuccessCriteria == "" {
			return false
		}
		ids[s.ID] = true
	}
	for _, s := range p.Steps {
		if s.OnSuccess != "complete" && !ids[s.OnSuccess] {
			return false
		}
		if s.OnDistress != "stop" && s.OnDistress != "pause" {
			return false
		}
		if s.OnDecline != "stop" && s.OnDecline != "pause" {
			return false
		}
		if s.OnStuck.Then != "" && s.OnStuck.Then != "stop" && s.OnStuck.Then != "pause" {
			return false
		}
		if s.OnStuck.After < 0 {
			return false
		}
		if s.OnStuck.Adapt != "" {
			found := false
			for _, a := range s.Adaptations {
				if a.ID == s.OnStuck.Adapt && a.Instruction != "" {
					found = true
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
func (a *Assets) Reference() c.BundleRef {
	raw, _ := json.Marshal(a.ref)
	var out c.BundleRef
	_ = json.Unmarshal(raw, &out)
	return out
}
func (a *Assets) SyntheticOnly() bool { return a.synthetic }
func (a *Assets) Protocol(id string) (bundle.Protocol, bool) {
	p, ok := a.protocols[id]
	if !ok {
		return p, false
	}
	raw, _ := json.Marshal(p)
	var out bundle.Protocol
	_ = json.Unmarshal(raw, &out)
	return out, true
}
