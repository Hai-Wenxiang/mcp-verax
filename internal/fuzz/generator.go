// Package fuzz generates schema-conforming and hostile tool arguments from
// JSON Schema (a pragmatic subset: type/properties/required/items/
// min-max/enum) and classifies crash/hang behaviour.
package fuzz

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
)

// Generator produces arguments for one tool from its inputSchema.
type Generator struct {
	rng *rand.Rand
	// EdgeFraction in [0,1]: probability of picking a dictionary value for
	// string properties instead of a benign one.
	EdgeFraction float64
}

// NewGenerator seeds deterministically.
func NewGenerator(seed int64, edgeFraction float64) *Generator {
	if edgeFraction < 0 {
		edgeFraction = 0
	}
	if edgeFraction > 1 {
		edgeFraction = 1
	}
	return &Generator{rng: rand.New(rand.NewSource(seed)), EdgeFraction: edgeFraction}
}

// Args generates one argument set for a schema. A nil/invalid schema yields
// a single hostile key-value probe.
func (g *Generator) Args(schema []byte) (map[string]any, error) {
	if len(schema) == 0 {
		return map[string]any{"input": g.stringValue()}, nil
	}
	var s struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil, fmt.Errorf("fuzz: decode schema: %w", err)
	}
	if len(s.Properties) == 0 {
		return map[string]any{"input": g.stringValue()}, nil
	}
	// Generate in sorted property order: map iteration is randomized per run,
	// and same-seed determinism is a core fuzzing guarantee.
	names := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make(map[string]any, len(s.Properties))
	for _, name := range names {
		v, err := g.value(s.Properties[name])
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
	// Required keys are always present above (properties drive generation);
	// non-required keys are included too — servers must tolerate them.
	return out, nil
}

func (g *Generator) value(raw json.RawMessage) (any, error) {
	var s struct {
		Type    string          `json:"type"`
		Enum    []any           `json:"enum"`
		Items   json.RawMessage `json:"items"`
		Minimum *float64        `json:"minimum"`
		Maximum *float64        `json:"maximum"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("fuzz: decode property: %w", err)
	}
	if len(s.Enum) > 0 {
		return s.Enum[g.rng.Intn(len(s.Enum))], nil
	}
	switch s.Type {
	case "string":
		return g.stringValue(), nil
	case "integer":
		return int(g.number(s.Minimum, s.Maximum)), nil
	case "number":
		return g.number(s.Minimum, s.Maximum), nil
	case "boolean":
		return g.rng.Intn(2) == 0, nil
	case "array":
		item := any(nil)
		if len(s.Items) > 0 {
			v, err := g.value(s.Items)
			if err != nil {
				return nil, err
			}
			item = v
		} else {
			item = g.stringValue()
		}
		return []any{item}, nil
	case "object", "":
		return map[string]any{"k": g.stringValue()}, nil
	default:
		return g.stringValue(), nil
	}
}

func (g *Generator) number(min, max *float64) float64 {
	lo, hi := 0.0, 100.0
	if min != nil {
		lo = *min
	}
	if max != nil {
		hi = *max
	}
	if hi < lo {
		hi = lo
	}
	return lo + g.rng.Float64()*(hi-lo)
}

func (g *Generator) stringValue() string {
	if g.rng.Float64() < g.EdgeFraction {
		return PickDictionary(g.rng)
	}
	return benignStrings[g.rng.Intn(len(benignStrings))]
}

var benignStrings = []string{"ok", "hello", "sample text", "123", "测试文本", "a@b.com"}
