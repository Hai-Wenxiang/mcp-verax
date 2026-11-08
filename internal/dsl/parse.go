package dsl

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse decodes and validates a suite definition. Unknown YAML fields are
// rejected (with line numbers where available) so DSL typos fail loudly.
func Parse(data []byte, source string) (*Suite, error) {
	var s Suite
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		var typeErr *yaml.TypeError
		if asTypeError(err, &typeErr) {
			var msgs []string
			for _, e := range typeErr.Errors {
				msgs = append(msgs, "  - "+strings.TrimSpace(e))
			}
			return nil, fmt.Errorf("%s: unknown or mistyped fields:\n%s", sourceName(source), strings.Join(msgs, "\n"))
		}
		return nil, fmt.Errorf("%s: parse YAML: %w", sourceName(source), err)
	}
	s.Source = source
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", sourceName(source), err)
	}
	return &s, nil
}

// ParseFile loads a suite from disk.
func ParseFile(path string) (*Suite, error) {
	data, err := osReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data, path)
}

func sourceName(s string) string {
	if s == "" {
		return "suite"
	}
	return s
}

func asTypeError(err error, target **yaml.TypeError) bool {
	te, ok := err.(*yaml.TypeError)
	if ok {
		*target = te
	}
	return ok
}

// splitQuoted splits a command line on whitespace, honouring single and
// double quotes (no shell features, no escapes beyond quotes).
func splitQuoted(s string) ([]string, error) {
	var parts []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unbalanced quotes")
	}
	flush()
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return parts, nil
}
