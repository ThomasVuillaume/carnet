package config

import (
	"errors"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

// validConfig is the smallest config validate accepts: the defaults plus the
// four required keys. Tests mutate one field of it to isolate one rejection.
func validConfig() Config {
	c := Defaults()
	c.Title = "Traversée du Vercors"
	c.Track = "track.gpx"
	c.StartDate = Date{raw: "2026-05-08"}
	c.EndDate = Date{raw: "2026-05-10"}
	return c
}

// reportedKeys extracts the faulty keys in report order. It fails the test when
// err is not a ValidationErrors, since that is the contract cmd/carnet relies on
// for exit code 2.
func reportedKeys(tb testing.TB, err error) []string {
	tb.Helper()
	if err == nil {
		return nil
	}
	var v ValidationErrors
	if !errors.As(err, &v) {
		tb.Fatalf("errors.As did not reach ValidationErrors, got %T: %v", err, err)
	}
	keys := make([]string, len(v))
	for i, e := range v {
		keys[i] = e.Key
	}
	return keys
}

// The defaults are the PRD §6.2 sample, and this table is the only thing
// keeping the two in step. A default drifting away from the documented example
// is silent otherwise: the file still loads, it just produces something else.
func TestDefaultsMatchPRD(t *testing.T) {
	t.Parallel()
	d := Defaults()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"photos_dir", d.PhotosDir, "photos"},
		{"map.width", d.Map.Width, 1280},
		{"map.height", d.Map.Height, 720},
		{"map.max_zoom", d.Map.MaxZoom, 15},
		{"map.padding", d.Map.Padding, 32},
		{"map.track_color", d.Map.TrackColor, "#c1121f"},
		{"map.track_width", d.Map.TrackWidth, 4},
		{"map.tile_source", d.Map.TileSource, "osm"},
		{"privacy.trim_start_m", d.Privacy.TrimStartM, 3000},
		{"privacy.trim_end_m", d.Privacy.TrimEndM, 3000},
		{"privacy.coordinate_precision", d.Privacy.CoordinatePrecision, 3},
		{"privacy.strip_exif", d.Privacy.StripExif, true},
		{"privacy.keep_exif_tags", d.Privacy.KeepExifTags, []string{
			"Make", "Model", "LensModel", "FocalLength", "FNumber", "ExposureTime", "ISOSpeedRatings",
		}},
		{"photos.clock_offset_s", d.Photos.ClockOffsetS, 0},
		{"photos.match_tolerance_s", d.Photos.MatchToleranceS, 120},
		{"photos.match_mode", d.Photos.MatchMode, "gps_then_time"},
		{"photos.divergence_warn_m", d.Photos.DivergenceWarnM, 100},
		{"photos.captions_file", d.Photos.CaptionsFile, "captions.yaml"},
		// Required keys carry no default: a value here would make an absent key
		// indistinguishable from a supplied one and defeat validate.
		{"title", d.Title, ""},
		{"start_date", d.StartDate.String(), ""},
		{"end_date", d.EndDate.String(), ""},
		{"track", d.Track, ""},
		// nil rather than an empty slice: the zero value round-trips through
		// YAML unchanged, an explicit [] does not.
		{"tags", d.Tags, []string(nil)},
		{"privacy.exclusion_zones", d.Privacy.ExclusionZones, []ExclusionZone(nil)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Errorf("got %#v, want %#v", tc.got, tc.want)
			}
		})
	}
}

// Defaults returns a fresh value on every call. A package-level var would share
// the KeepExifTags backing array between loads, and the decoder reuses that
// array in place, so two Loads in one process would corrupt each other.
func TestDefaultsAreIndependentAcrossCalls(t *testing.T) {
	t.Parallel()

	first := Defaults()
	first.Privacy.KeepExifTags[0] = "Mutated"
	first.Map.Width = 1

	second := Defaults()
	if got := second.Privacy.KeepExifTags[0]; got != "Make" {
		t.Errorf("KeepExifTags[0] = %q, want %q: the slice is shared between calls", got, "Make")
	}
	if got := second.Map.Width; got != 1280 {
		t.Errorf("Map.Width = %d, want 1280", got)
	}
}

// The defaults must sit inside the bounds validate enforces, with the required
// keys supplied. A default resting on its own bound would make the bound test
// meaningless.
func TestDefaultsPassValidation(t *testing.T) {
	t.Parallel()

	if err := validate(validConfig()); err != nil {
		t.Errorf("validate(validConfig()) = %v, want nil", err)
	}
}

// Date takes the scalar's raw text whatever tag the YAML resolver assigned, so
// both spellings of a day load. The table discriminates tag resolution, not
// date correctness: judging the format is validate's job, and cases that are
// nonsense as dates still decode here.
func TestDateUnmarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  string
		want    string
		wantErr bool
	}{
		{"bare date resolves to !!timestamp", "d: 2026-05-08", "2026-05-08", false},
		{"quoted date resolves to !!str", `d: "2026-05-08"`, "2026-05-08", false},
		{"single-quoted date", "d: '2026-05-08'", "2026-05-08", false},
		{"absent key leaves the zero Date", "other: 1", "", false},
		{"explicit empty string", `d: ""`, "", false},
		{"malformed date still decodes", "d: 08/05/2026", "08/05/2026", false},
		{"out-of-range date still decodes", `d: "2026-13-45"`, "2026-13-45", false},
		{"sequence has no text to record", "d: [2026, 5, 8]", "", true},
		{"mapping has no text to record", "d: {year: 2026}", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var holder struct {
				Other int  `yaml:"other"`
				D     Date `yaml:"d"`
			}
			err := yaml.Unmarshal([]byte(tc.source+"\n"), &holder)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Unmarshal error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if got := holder.D.String(); got != tc.want {
				t.Errorf("Date.String() = %q, want %q", got, tc.want)
			}
		})
	}
}
