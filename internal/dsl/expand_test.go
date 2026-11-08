package dsl

import (
	"strings"
	"testing"
)

func lookup(env map[string]string) LookupFunc {
	return func(name string) (string, bool) {
		v, ok := env[name]
		return v, ok
	}
}

func TestExpandEnvCommandAndEnv(t *testing.T) {
	s, err := Parse([]byte(validSuite), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// inject a placeholder into the command
	s.Target.Command = "npx -y ${PKG_NAME} --port ${PORT}"
	s.Target.Env["TOKEN"] = "tok-${SECRET}"

	out, err := s.ExpandEnv(lookup(map[string]string{
		"PKG_NAME": "@scope/server",
		"PORT":     "8080",
		"SECRET":   "abc",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Target.Command != "npx -y @scope/server --port 8080" {
		t.Errorf("command = %q", out.Target.Command)
	}
	if out.Target.Env["TOKEN"] != "tok-abc" {
		t.Errorf("env TOKEN = %q", out.Target.Env["TOKEN"])
	}
	parts, err := out.Target.CommandParts()
	if err != nil || parts[2] != "@scope/server" {
		t.Errorf("expanded command should still split: %v %v", parts, err)
	}
}

func TestExpandEnvCaseArgsAndEnv(t *testing.T) {
	s, err := Parse([]byte(validSuite), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s.Cases[1].Args = map[string]any{"msg": "user-${USER_ID}", "count": 3, "flag": true}
	s.Cases[1].Env = map[string]string{"MODE": "${VERAX_MODE}"}

	out, err := s.ExpandEnv(lookup(map[string]string{"USER_ID": "42", "VERAX_MODE": "strict"}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Cases[1].Args["msg"] != "user-42" {
		t.Errorf("expanded arg = %v", out.Cases[1].Args["msg"])
	}
	if out.Cases[1].Args["count"] != 3 || out.Cases[1].Args["flag"] != true {
		t.Errorf("non-string args must pass through: %+v", out.Cases[1].Args)
	}
	if out.Cases[1].Env["MODE"] != "strict" {
		t.Errorf("expanded case env = %v", out.Cases[1].Env)
	}
}

func TestExpandEnvUndefinedVarIsError(t *testing.T) {
	s, err := Parse([]byte(validSuite), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s.Target.Command = "run ${MISSING_THING}"
	_, err = s.ExpandEnv(lookup(map[string]string{}))
	if err == nil || !strings.Contains(err.Error(), "MISSING_THING") {
		t.Fatalf("undefined var must error naming it: %v", err)
	}
	if !strings.Contains(err.Error(), "target.command") {
		t.Errorf("error should carry location: %v", err)
	}
}

func TestExpandEnvBareDollarUntouched(t *testing.T) {
	s, err := Parse([]byte(validSuite), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s.Target.Command = "sh -c 'echo $HOME is yours'"
	out, err := s.ExpandEnv(lookup(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Target.Command != "sh -c 'echo $HOME is yours'" {
		t.Errorf("bare $VAR must stay untouched: %q", out.Target.Command)
	}
}

func TestExpandEnvDoesNotMutateOriginal(t *testing.T) {
	s, err := Parse([]byte(validSuite), "v.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s.Target.Command = "run ${X}"
	out, err := s.ExpandEnv(lookup(map[string]string{"X": "1"}))
	if err != nil {
		t.Fatal(err)
	}
	if s.Target.Command != "run ${X}" {
		t.Errorf("original suite mutated: %q", s.Target.Command)
	}
	if out.Target.Command != "run 1" {
		t.Errorf("copy not expanded: %q", out.Target.Command)
	}
}
