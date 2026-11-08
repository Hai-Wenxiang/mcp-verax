package fuzz

import (
	"encoding/json"
	"strings"
	"testing"
)

const echoSchema = `{
  "type": "object",
  "properties": {
    "msg":   {"type": "string", "minLength": 1},
    "count": {"type": "integer", "minimum": 1, "maximum": 10},
    "flag":  {"type": "boolean"},
    "mode":  {"enum": ["fast", "slow"]},
    "tags":  {"type": "array", "items": {"type": "string"}},
    "meta":  {"type": "object"}
  },
  "required": ["msg"]
}`

func TestArgsValidSchemaTypes(t *testing.T) {
	g := NewGenerator(42, 0) // pure benign mode
	args, err := g.Args([]byte(echoSchema))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := args["msg"].(string); !ok {
		t.Errorf("msg should be string: %#v", args["msg"])
	}
	if c, ok := args["count"].(int); !ok || c < 1 || c > 10 {
		t.Errorf("count should be int within [1,10]: %#v", args["count"])
	}
	if _, ok := args["flag"].(bool); !ok {
		t.Errorf("flag should be bool: %#v", args["flag"])
	}
	if m, ok := args["mode"].(string); !ok || (m != "fast" && m != "slow") {
		t.Errorf("mode should be enum value: %#v", args["mode"])
	}
	tags, ok := args["tags"].([]any)
	if !ok || len(tags) == 0 {
		t.Errorf("tags should be non-empty array: %#v", args["tags"])
	}
	if _, ok := args["meta"].(map[string]any); !ok {
		t.Errorf("meta should be object: %#v", args["meta"])
	}
}

func TestArgsDeterministicWithSeed(t *testing.T) {
	a, _ := NewGenerator(7, 0.5).Args([]byte(echoSchema))
	b, _ := NewGenerator(7, 0.5).Args([]byte(echoSchema))
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("seeded generator must be deterministic:\n%s\n%s", ja, jb)
	}
	c, _ := NewGenerator(8, 0.5).Args([]byte(echoSchema))
	jc, _ := json.Marshal(c)
	if string(ja) == string(jc) {
		t.Error("different seeds should (almost surely) differ")
	}
}

func TestEdgeModeInjectsDictionary(t *testing.T) {
	g := NewGenerator(1, 1) // always dictionary
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		args, err := g.Args([]byte(echoSchema))
		if err != nil {
			t.Fatal(err)
		}
		seen[args["msg"].(string)] = true
	}
	matched := 0
	for _, d := range Dictionary {
		if seen[d] {
			matched++
		}
	}
	if matched < 3 {
		t.Errorf("edge mode should surface many dictionary entries, got %d", matched)
	}
}

func TestArgsEmptySchemaFallback(t *testing.T) {
	g := NewGenerator(3, 0)
	args, err := g.Args(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := args["input"]; !ok {
		t.Errorf("nil schema should probe 'input': %#v", args)
	}
}

func TestDictionaryHostility(t *testing.T) {
	joined := strings.Join(Dictionary, "\x00")
	for _, want := range []string{"../../etc/passwd", "Ignore previous instructions", "DROP TABLE", "\u0000null-byte"} {
		if !strings.Contains(joined, want) {
			t.Errorf("dictionary missing %q", want)
		}
	}
	// policy: inert payloads — real-looking URLs and credential shapes banned;
	// example.invalid/localhost-style documentation hosts are allowed
	for _, banned := range []string{"https://", "http://example.com", "sk-", "AKIA", "ghp_"} {
		if strings.Contains(joined, banned) {
			t.Errorf("dictionary must stay inert, contains %q", banned)
		}
	}
}
