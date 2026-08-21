package config

import (
	"fmt"
	"strings"
)

// InvalidConfigError names the key at fault. Every rejection validate reports
// carries one; a document the decoder refuses outright carries none and exits
// with code 1 instead. cmd/carnet detects this type with errors.As, so the exit
// code follows from a type rather than from a parsed message.
//
// Key is the YAML path the user typed ("map.max_zoom"), never the Go field
// name: the message has to point back into their file. Value carries what the
// file held, so the reader sees the offending input without reopening it.
type InvalidConfigError struct {
	Key    string
	Value  any
	Reason string
}

// Error renders one rejection as a single line.
//
// Value stays nil when the file offers nothing to quote back: the key is
// absent, or it carries a value that renders empty. Echoing either would print
// "invalid value :" and send the reader looking for input that was never there.
//
// Pointer receiver, so every errors.As target is **InvalidConfigError, tests
// included.
func (e *InvalidConfigError) Error() string {
	if e.Value == nil {
		return fmt.Sprintf("%s: %s", e.Key, e.Reason)
	}
	return fmt.Sprintf("%s: invalid value %v: %s", e.Key, e.Value, e.Reason)
}

// ValidationErrors carries every key validate rejected. Entries follow the
// order the checks run in, never map order, so the report is byte-identical
// across runs.
type ValidationErrors []*InvalidConfigError

// Unwrap in its plural form is what keeps errors.As reaching the entries.
// Without it the slice is opaque and cmd/carnet loses exit code 2.
func (v ValidationErrors) Unwrap() []error {
	errs := make([]error, len(v))
	for i, e := range v {
		errs[i] = e
	}
	return errs
}

// Err converts to error, and to nil when empty. Returning v directly would
// hand back a non-nil interface holding a nil slice, which callers see as a
// failure on a valid file.
func (v ValidationErrors) Err() error {
	if len(v) == 0 {
		return nil
	}
	return v
}

// Error opens on a count, then gives one line per entry, rendered through
// InvalidConfigError.Error: the line format lives in one place and a lone
// rejection reads the same as one buried in a list. No
// file name: doc.go keeps this package on an io.Reader, and cmd/carnet knows
// which path it opened. No trailing newline either, since the caller printing
// the error supplies it.
func (v ValidationErrors) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d invalid key", len(v))
	if len(v) > 1 {
		b.WriteString("s")
	}
	b.WriteString(":")
	for _, e := range v {
		b.WriteString("\n  - ")
		b.WriteString(e.Error())
	}
	return b.String()
}
