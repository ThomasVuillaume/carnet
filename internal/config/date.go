package config

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

// Date is a calendar day exactly as the file spells it, kept unparsed.
//
// The YAML resolver tags a bare 2026-05-08 as !!timestamp and a quoted one as
// !!str, and it decides before the decoder looks at the Go type. Reading the
// node's raw text accepts both spellings, and defers judgement to validate so
// a malformed date stays an InvalidConfigError on exit code 2 instead of
// becoming a decoder failure on exit code 1.
type Date struct {
	raw string
}

// UnmarshalYAML records the scalar without interpreting it. A non-scalar node
// carries no text to record, so it fails here rather than reaching validate as
// an indistinguishable empty date. The decoder prefixes the line number, so
// the message must not repeat it.
func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return errors.New("a date must be a single value")
	}
	d.raw = n.Value
	return nil
}

// String returns the day as written, empty when the key was absent.
func (d Date) String() string {
	return d.raw
}
