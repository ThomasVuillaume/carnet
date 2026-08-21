package config

import (
	"reflect"
	"strings"
	"testing"
)

// Each row starts from a config validate accepts and breaks exactly one thing,
// so a reported key that is not in wantKeys is a false positive rather than a
// cascade. Rows carry the bound values on both sides of each limit: PRD §6.2
// states them as inclusive, and off-by-one there silently widens the contract.
func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*Config)
		wantKeys []string
	}{
		{"the base config is accepted", func(*Config) {}, nil},

		{"title absent", func(c *Config) { c.Title = "" }, []string{"title"}},
		{"track absent", func(c *Config) { c.Track = "" }, []string{"track"}},
		{"start_date absent", func(c *Config) { c.StartDate = Date{} }, []string{"start_date"}},
		{"end_date absent", func(c *Config) { c.EndDate = Date{} }, []string{"end_date"}},

		{"start_date in another notation", func(c *Config) { c.StartDate = Date{raw: "08/05/2026"} }, []string{"start_date"}},
		{"end_date out of range", func(c *Config) { c.EndDate = Date{raw: "2026-13-45"} }, []string{"end_date"}},
		{"end_date before start_date", func(c *Config) { c.EndDate = Date{raw: "2026-05-07"} }, []string{"end_date"}},
		{"a one-day trip repeats the date", func(c *Config) { c.EndDate = Date{raw: "2026-05-08"} }, nil},
		// An unparsable start_date leaves start zero, and the ordering check must
		// stay quiet rather than report a second key off a value it never got.
		{"unparsable start_date does not trigger the ordering check", func(c *Config) {
			c.StartDate = Date{raw: "nope"}
		}, []string{"start_date"}},

		{"map.width below the floor", func(c *Config) { c.Map.Width = 319 }, []string{"map.width"}},
		{"map.width on the floor", func(c *Config) { c.Map.Width = 320 }, nil},
		{"map.width on the ceiling", func(c *Config) { c.Map.Width = 4096 }, nil},
		{"map.width above the ceiling", func(c *Config) { c.Map.Width = 4097 }, []string{"map.width"}},
		{"map.height below the floor", func(c *Config) { c.Map.Height = 319 }, []string{"map.height"}},
		{"map.height above the ceiling", func(c *Config) { c.Map.Height = 4097 }, []string{"map.height"}},

		{"map.max_zoom on the ceiling", func(c *Config) { c.Map.MaxZoom = 18 }, nil},
		{"map.max_zoom above the ceiling", func(c *Config) { c.Map.MaxZoom = 19 }, []string{"map.max_zoom"}},

		{"coordinate_precision below zero", func(c *Config) { c.Privacy.CoordinatePrecision = -1 }, []string{"privacy.coordinate_precision"}},
		{"coordinate_precision at zero", func(c *Config) { c.Privacy.CoordinatePrecision = 0 }, nil},
		{"coordinate_precision on the ceiling", func(c *Config) { c.Privacy.CoordinatePrecision = 6 }, nil},
		{"coordinate_precision above the ceiling", func(c *Config) { c.Privacy.CoordinatePrecision = 7 }, []string{"privacy.coordinate_precision"}},

		{"exclusion radius at zero", func(c *Config) {
			c.Privacy.ExclusionZones = []ExclusionZone{{Lat: 43.6, Lon: 1.44, RadiusM: 0, Label: "home"}}
		}, []string{"privacy.exclusion_zones[0].radius_m"}},
		{"exclusion radius negative", func(c *Config) {
			c.Privacy.ExclusionZones = []ExclusionZone{{Lat: 43.6, Lon: 1.44, RadiusM: -1}}
		}, []string{"privacy.exclusion_zones[0].radius_m"}},
		// The index, not the label, identifies the entry: labels are optional
		// and may repeat, so only the position always points at one zone.
		{"the faulty zone is named by its index", func(c *Config) {
			c.Privacy.ExclusionZones = []ExclusionZone{
				{RadiusM: 3000, Label: "home"},
				{RadiusM: 0, Label: "home"},
			}
		}, []string{"privacy.exclusion_zones[1].radius_m"}},

		{"match_mode gps", func(c *Config) { c.Photos.MatchMode = "gps" }, nil},
		{"match_mode time", func(c *Config) { c.Photos.MatchMode = "time" }, nil},
		// Defaults fills the key, so only an explicit empty value lands here.
		{"match_mode explicitly emptied", func(c *Config) { c.Photos.MatchMode = "" }, []string{"photos.match_mode"}},
		{"match_mode unknown", func(c *Config) { c.Photos.MatchMode = "timestamp" }, []string{"photos.match_mode"}},
		{"match_mode near-miss", func(c *Config) { c.Photos.MatchMode = "gps_then_tim" }, []string{"photos.match_mode"}},
		// The decision this row pins down: the comparison is case-sensitive, as
		// YAML scalars are. Nothing else in the code states it.
		{"match_mode is case-sensitive", func(c *Config) { c.Photos.MatchMode = "GPS_then_time" }, []string{"photos.match_mode"}},

		// Every fault at once, in the order the checks run: the report exists to
		// spare the reader five successive runs.
		{"faults accumulate in check order", func(c *Config) {
			c.Title = ""
			c.Track = ""
			c.StartDate = Date{}
			c.Map.MaxZoom = 30
			c.Privacy.CoordinatePrecision = 9
		}, []string{"title", "track", "start_date", "map.max_zoom", "privacy.coordinate_precision"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := validConfig()
			tc.mutate(&c)

			got := reportedKeys(t, validate(c))
			if !reflect.DeepEqual(got, tc.wantKeys) {
				t.Errorf("reported keys = %v, want %v\nreport:\n%v", got, tc.wantKeys, validate(c))
			}
		})
	}
}

// The report is part of the deterministic output contract of PRD §4.3: the same
// faulty config must produce the same bytes, so no check may derive its order
// from map iteration.
func TestValidateReportIsStable(t *testing.T) {
	t.Parallel()

	c := validConfig()
	c.Title = ""
	c.Map.MaxZoom = 30
	c.Privacy.ExclusionZones = []ExclusionZone{{RadiusM: 0}, {RadiusM: -5}}

	first := validate(c).Error()
	for range 20 {
		if got := validate(c).Error(); got != first {
			t.Fatalf("report changed between runs:\n%s\n---\n%s", first, got)
		}
	}
}

// A missing required key has no offending input to quote, so Reason stands
// alone. A supplied one is echoed back, since pointing at the value is what
// lets the reader find it in the file.
func TestInvalidConfigErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"absent key omits the value", func(c *Config) { c.Title = "" }, "title: required"},
		{"supplied key echoes the value", func(c *Config) { c.Map.MaxZoom = 19 }, "map.max_zoom: invalid value 19: must be at most 18"},
		{"a Date echoes as written", func(c *Config) { c.StartDate = Date{raw: "08/05/2026"} }, "start_date: invalid value 08/05/2026: must be a date in YYYY-MM-DD form"},
		// The allowed set is spelled out: the report is the only place carnet
		// tells the reader what match_mode accepts.
		{"an unknown enum lists the allowed values", func(c *Config) { c.Photos.MatchMode = "timestamp" },
			"photos.match_mode: invalid value timestamp: must be one of gps, time, gps_then_time"},
		// An emptied enum echoes nothing: quoting "" would print a dangling
		// "invalid value :" with no input between the colons.
		{"an emptied enum echoes no value", func(c *Config) { c.Photos.MatchMode = "" },
			"photos.match_mode: must be one of gps, time, gps_then_time"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := validConfig()
			tc.mutate(&c)

			got := validate(c).Error()
			if !strings.Contains(got, tc.want) {
				t.Errorf("report does not contain %q:\n%s", tc.want, got)
			}
		})
	}
}

// Singular and plural are the whole reason ValidationErrors writes its own
// Error rather than deferring to errors.Join, so the count line is worth a row
// each. The empty case must be nil, not an error wrapping an empty slice.
func TestValidationErrorsHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs ValidationErrors
		want string
	}{
		{"empty yields no error at all", nil, ""},
		{"one fault is singular", ValidationErrors{
			{Key: "title", Reason: "required"},
		}, "1 invalid key:\n  - title: required"},
		{"two faults are plural", ValidationErrors{
			{Key: "title", Reason: "required"},
			{Key: "map.max_zoom", Value: 19, Reason: "must be at most 18"},
		}, "2 invalid keys:\n  - title: required\n  - map.max_zoom: invalid value 19: must be at most 18"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.errs.Err()
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Err() = %v, want nil: a nil slice returned as error is a non-nil interface", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Err() = nil, want an error")
			}
			if got := err.Error(); got != tc.want {
				t.Errorf("Error() =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
