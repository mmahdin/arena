package contextstate

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Policy captures only the WP-03 artifacts consumed by Context/State. Each
// call pins one immutable bundle handle; never read knowledge/ directly at
// runtime or invent absent clinical configuration.
type Policy struct {
	Ref        c.BundleRef
	Handle     ports.Bundle // pinned throughout this operation
	Problem    bundle.ProblemUnderstanding
	Thresholds bundle.Thresholds
	Gate       bundle.PolicyGate
	Budgets    bundle.Budgets
	Synthetic  bool
}

func LoadPolicy(ctx context.Context, reg ports.BundleRegistry, userID c.ID) (Policy, error) {
	var out Policy
	if reg == nil {
		return out, fmt.Errorf("WP-08 requires a Knowledge Bundle registry")
	}
	pinned, err := reg.Active(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("pin knowledge bundle: %w", err)
	}
	if pinned == nil || pinned.Reference().Version == "" {
		return out, fmt.Errorf("no pinned bundle")
	}
	out.Handle = pinned
	out.Ref = pinned.Reference()
	if s, ok := pinned.(interface{ SyntheticOnly() bool }); ok {
		out.Synthetic = s.SyntheticOnly()
	}
	for _, v := range []struct {
		path   string
		target any
	}{
		{"schemas/problem_understanding.yaml", &out.Problem},
		{"config/thresholds.yaml", &out.Thresholds},
		{"config/policy_gate.yaml", &out.Gate},
		{"config/budgets.yaml", &out.Budgets},
	} {
		raw, err := pinned.Artifact(ctx, v.path)
		if err != nil {
			return Policy{}, fmt.Errorf("load %s: %w", v.path, err)
		}
		if err = decodeStrict(raw, v.target); err != nil {
			return Policy{}, fmt.Errorf("decode %s: %w", v.path, err)
		}
	}
	if out.Problem.Schema != "problem.v1" || out.Thresholds.Schema != "thresholds.v1" || out.Gate.Schema != "policy_gate.v1" || out.Budgets.Schema != "budgets.v1" {
		return Policy{}, fmt.Errorf("unsupported context/state bundle schema")
	}
	return out, nil
}

// ProtocolGraph is loaded from a pinned WP-03 artifact only when a protocol
// transition is proposed; it is not a protocol engine or clinical rule source.
func ProtocolGraph(ctx context.Context, b ports.Bundle, protocolID string) (bundle.Protocol, error) {
	var p bundle.Protocol
	if b == nil || protocolID == "" {
		return p, fmt.Errorf("missing protocol bundle")
	}
	// Avoid path traversal even when an ID originated in persisted state.
	for _, ch := range protocolID {
		if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_') {
			return p, fmt.Errorf("invalid protocol ID")
		}
	}
	raw, err := b.Artifact(ctx, "protocols/"+protocolID+".yaml")
	if err != nil {
		return p, err
	}
	if err = decodeStrict(raw, &p); err != nil {
		return p, err
	}
	if p.Schema != "protocol.v1" || p.ID != protocolID {
		return p, fmt.Errorf("unrecognized protocol graph")
	}
	return p, nil
}

func decodeStrict(raw []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("more than one YAML document")
		}
		return err
	}
	return nil
}
