package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// expandRe matches the only interpolation form the DSL supports: ${VAR}.
// A bare $VAR is intentionally left untouched.
var expandRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// LookupFunc resolves an environment variable name to its value.
type LookupFunc func(name string) (string, bool)

// ExpandEnv returns a copy of the suite with ${VAR} placeholders expanded in
// target.command, target env/header values, per-case env values and string
// case args. Undefined variables are an error naming the variable and its
// location — silent empty expansion hides CI mistakes.
func (s *Suite) ExpandEnv(lookup LookupFunc) (*Suite, error) {
	out := *s

	if s.Target.Command != "" {
		v, err := expandString(s.Target.Command, "target.command", lookup)
		if err != nil {
			return nil, err
		}
		out.Target.Command = v
	}
	out.Target.Env, _ = expandStringMap(s.Target.Env, "target.env", lookup)
	out.Target.Headers, _ = expandStringMap(s.Target.Headers, "target.headers", lookup)

	cases := make([]Case, len(s.Cases))
	for i, c := range s.Cases {
		env2, err := expandStringMap(c.Env, fmt.Sprintf("case[%d] %q env", i, c.Name), lookup)
		if err != nil {
			return nil, err
		}
		c.Env = env2
		if len(c.Args) > 0 {
			args := make(map[string]any, len(c.Args))
			for k, v := range c.Args {
				if sv, ok := v.(string); ok {
					ev, err := expandString(sv, fmt.Sprintf("case[%d] %q args.%s", i, c.Name, k), lookup)
					if err != nil {
						return nil, err
					}
					v = ev
				}
				args[k] = v
			}
			c.Args = args
		}
		cases[i] = c
	}
	out.Cases = cases
	return &out, nil
}

func expandString(s, where string, lookup LookupFunc) (string, error) {
	var missing []string
	out := expandRe.ReplaceAllStringFunc(s, func(m string) string {
		name := m[2 : len(m)-1]
		if v, ok := lookup(name); ok {
			return v
		}
		missing = append(missing, name)
		return m
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("%s: undefined env var %s", where, strings.Join(missing, ", "))
	}
	return out, nil
}

func expandStringMap(m map[string]string, where string, lookup LookupFunc) (map[string]string, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		ev, err := expandString(v, where+"."+k, lookup)
		if err != nil {
			return nil, err
		}
		out[k] = ev
	}
	return out, nil
}
