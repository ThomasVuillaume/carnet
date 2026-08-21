package config

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Which of the two failure paths a source takes decides its CLI exit code, so
// the tests assert the path and not only the fact of failing.
type failure int

const (
	succeeds     failure = iota // exit 0
	failsToParse                // exit 1: not a faulty key, a file the decoder cannot read
	failsToPass                 // exit 2: an InvalidConfigError cmd/carnet can detect
)

// The sample of PRD §6.2, verbatim and unquoted. Loading it is the contract the
// documentation makes to the reader, and the dates in it are the trap: the YAML
// resolver tags a bare 2026-05-08 as !!timestamp.
const prdSample = `
title: "Traversée du Vercors"
start_date: 2026-05-08
end_date: 2026-05-10
summary: "Trois jours entre Grenoble et Die, cols et gorges."
track: track.gpx
photos_dir: photos
tags: [moto, vercors, alpes]

map:
  width: 1280
  height: 720
  max_zoom: 15
  padding: 32
  track_color: "#c1121f"
  track_width: 4
  tile_source: osm

privacy:
  exclusion_zones:
    - { lat: 43.6045, lon: 1.4442, radius_m: 3000, label: home }
  trim_start_m: 3000
  trim_end_m: 3000
  coordinate_precision: 3
  strip_exif: true
  keep_exif_tags: [Make, Model, LensModel, FocalLength, FNumber, ExposureTime, ISOSpeedRatings]

photos:
  clock_offset_s: 0
  match_tolerance_s: 120
  match_mode: gps_then_time
  divergence_warn_m: 100
  captions_file: captions.yaml
`

// The minimum a source needs to reach validate cleanly, for rows that break one
// unrelated thing.
const minimalSample = `
title: T
start_date: 2026-05-08
end_date: 2026-05-08
track: track.gpx
`

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		want     failure
		wantKeys []string // failsToPass only
	}{
		{"the PRD §6.2 sample loads as written", prdSample, succeeds, nil},
		{"quoted dates load too", strings.ReplaceAll(minimalSample, "2026-05-08", `"2026-05-08"`), succeeds, nil},

		// An empty file is not a syntax error: every required key is simply
		// missing, and naming them beats reporting the EOF.
		{"an empty document names the required keys", "", failsToPass,
			[]string{"title", "track", "start_date", "end_date"}},
		{"a comment-only document behaves the same", "# nothing here\n", failsToPass,
			[]string{"title", "track", "start_date", "end_date"}},

		{"a bound violation is a faulty key", minimalSample + "map:\n  max_zoom: 22\n", failsToPass,
			[]string{"map.max_zoom"}},

		// The reason KnownFields is on: a singular exclusion_zone would decode
		// into nothing and publish the coordinates the zone existed to hide.
		{"a misspelled privacy key is refused, not ignored",
			minimalSample + "privacy:\n  exclusion_zone: []\n", failsToParse, nil},
		{"a misspelled top-level key is refused", minimalSample + "titel: T\n", failsToParse, nil},
		{"a misspelled nested key is refused", minimalSample + "map:\n  max_zom: 12\n", failsToParse, nil},

		{"unreadable YAML fails to parse", "title: \"unterminated\n", failsToParse, nil},
		{"a date that is not a scalar fails to parse",
			"title: T\ntrack: t.gpx\nstart_date: [2026, 5, 8]\nend_date: 2026-05-08\n", failsToParse, nil},
		{"a string where a number belongs fails to parse",
			minimalSample + "map:\n  width: wide\n", failsToParse, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(strings.NewReader(tc.source))

			if tc.want == succeeds {
				if err != nil {
					t.Fatalf("Load() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Load() = nil, want an error")
			}
			// Nothing partially decoded escapes: a caller that skips the error
			// check would otherwise run the privacy pipeline on unvalidated
			// bounds.
			if !reflect.DeepEqual(cfg, Config{}) {
				t.Errorf("Load() returned %#v on error, want the zero Config", cfg)
			}

			var target *InvalidConfigError
			switch reached := errors.As(err, &target); tc.want {
			case failsToParse:
				if reached {
					t.Fatalf("decode failure surfaced as an InvalidConfigError, which would send exit code 2: %v", err)
				}
			case failsToPass:
				if !reached {
					t.Fatalf("validation failure did not surface as an InvalidConfigError, which would send exit code 1: %v", err)
				}
				if got := reportedKeys(t, err); !reflect.DeepEqual(got, tc.wantKeys) {
					t.Errorf("reported keys = %v, want %v", got, tc.wantKeys)
				}
			}
		})
	}
}

// An absent key keeps its default, an explicit key overwrites it even when it
// carries zero. The privacy keys are where the distinction has teeth: 0 means
// no trimming at all, absence means 3000 m.
func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		got    func(Config) any
		want   any
	}{
		{"an absent block keeps every default", minimalSample,
			func(c Config) any { return c.Map.Padding }, 32},
		{"an absent key inside a supplied block keeps its default",
			minimalSample + "map:\n  width: 800\n",
			func(c Config) any { return c.Map.Height }, 720},
		{"a supplied key overwrites its default",
			minimalSample + "map:\n  width: 800\n",
			func(c Config) any { return c.Map.Width }, 800},
		{"an explicit zero overwrites a non-zero default",
			minimalSample + "privacy:\n  trim_start_m: 0\n",
			func(c Config) any { return c.Privacy.TrimStartM }, 0},
		{"an absent trim keeps 3000 m",
			minimalSample + "privacy:\n  trim_end_m: 500\n",
			func(c Config) any { return c.Privacy.TrimStartM }, 3000},
		{"an explicit false overwrites a true default",
			minimalSample + "privacy:\n  strip_exif: false\n",
			func(c Config) any { return c.Privacy.StripExif }, false},
		{"a supplied list replaces the default list entirely",
			minimalSample + "privacy:\n  keep_exif_tags: [Make]\n",
			func(c Config) any { return c.Privacy.KeepExifTags }, []string{"Make"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := Load(strings.NewReader(tc.source))
			if err != nil {
				t.Fatalf("Load() = %v, want nil", err)
			}
			if got := tc.got(cfg); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// PRD §4.3: the same input must produce the same output. The risk here is not
// the values but the memory behind them — a shared defaults slice would be
// truncated and refilled in place by the decoder, so the second Load would see
// what the first one left.
func TestLoadIsRepeatable(t *testing.T) {
	t.Parallel()

	source := prdSample + "\n"

	first, err := Load(strings.NewReader(source))
	if err != nil {
		t.Fatalf("first Load() = %v", err)
	}
	first.Privacy.KeepExifTags[0] = "Mutated"

	second, err := Load(strings.NewReader(source))
	if err != nil {
		t.Fatalf("second Load() = %v", err)
	}
	if got := second.Privacy.KeepExifTags[0]; got != "Make" {
		t.Errorf("KeepExifTags[0] = %q after a previous Load mutated its own copy, want %q", got, "Make")
	}

	third, err := Load(strings.NewReader(source))
	if err != nil {
		t.Fatalf("third Load() = %v", err)
	}
	if !reflect.DeepEqual(second, third) {
		t.Error("two Loads of the same source produced different configs")
	}
}

// Load takes a reader and never closes it: the caller owns the file, which is
// what lets doc.go promise this package touches no filesystem.
func TestLoadLeavesTheReaderOpen(t *testing.T) {
	t.Parallel()

	r := &countingCloser{Reader: strings.NewReader(prdSample)}
	if _, err := Load(r); err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if r.closes != 0 {
		t.Errorf("Load closed the reader %d times, want 0", r.closes)
	}
}

type countingCloser struct {
	*strings.Reader
	closes int
}

func (c *countingCloser) Close() error {
	c.closes++
	return nil
}
