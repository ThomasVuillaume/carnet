package config

import (
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v4"
)

// Load reads carnet.yaml from r, fills in the defaults and validates the
// result. It takes a reader rather than a path so the caller owns the file: r
// is never closed here, and the messages carry no file name.
//
// Unknown keys are rejected. A misspelled "exclusion_zone" would otherwise
// decode into nothing and publish the coordinates the zone existed to hide.
//
// On any error the returned Config is the zero value, never the partially
// decoded one: a caller that forgets to check would otherwise run the privacy
// pipeline on unvalidated bounds.
func Load(r io.Reader) (Config, error) {
	cfg := Defaults()

	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	// An empty document yields io.EOF and leaves the defaults untouched.
	// validate then names every missing required key, which reads better than
	// reporting the EOF itself.
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("decoding carnet.yaml: %w", err)
	}

	// Returned unwrapped: ValidationErrors opens with its own count, and a
	// prefix would decorate only the first of its lines.
	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
