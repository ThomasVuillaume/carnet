package config

import (
	"fmt"
	"time"
)

// Bounds of PRD §6.2. Past zoom 18 public tile ladders mostly stop
// (https://wiki.openstreetmap.org/wiki/Zoom_levels). Below 320 px the
// attribution no longer fits; above 4096 the tile count and the PNG both stop
// being reasonable. Six decimals is about 11 cm, past which the digits are
// noise the rounding key exists to remove.
const (
	maxZoomCeiling         = 18
	minMapSide             = 320
	maxMapSide             = 4096
	maxCoordinatePrecision = 6
)

// Go layouts are the reference time itself, Mon Jan 2 15:04:05 MST 2006, not a
// strftime pattern. Anything the parser fails to recognise turns into a
// literal, so a wrong layout compiles and parses nothing.
const dateLayout = "2006-01-02"

// The three modes of PRD §6.5, spelled for the reader of the report: it is the
// only place carnet tells them what is allowed. Kept as a string because Go has
// no constant slice, and a package-level var would be mutable global state.
// Two other copies follow any change here: the switch below, which carries the
// set itself, and the doc comment on Photos.MatchMode.
const matchModes = "gps, time, gps_then_time"

// validate reports every rejected key rather than stopping at the first: the
// check command is worth far more when it lists the five faulty keys than when
// it hides four of them. Checks run in a fixed order — required keys, then map,
// privacy, photos — so the report is byte-identical across runs and the reader
// meets the blocking faults before the tuning ones.
//
// Value is left nil on a missing required key: there is no offending input to
// show, only an absence.
func validate(c Config) error {
	var errs ValidationErrors

	if c.Title == "" {
		errs = append(errs, &InvalidConfigError{Key: "title", Reason: "required"})
	}
	if c.Track == "" {
		errs = append(errs, &InvalidConfigError{Key: "track", Reason: "required"})
	}

	// Emptiness and format are tested in sequence, not chained: a failed Parse
	// yields both an error and the zero Time, so a chain would name the same key
	// twice. A date left zero here means "not parsed", which is what the
	// ordering check below reads it as.
	var start, end time.Time
	if c.StartDate.String() == "" {
		errs = append(errs, &InvalidConfigError{Key: "start_date", Reason: "required"})
	} else if t, err := time.Parse(dateLayout, c.StartDate.String()); err != nil {
		errs = append(errs, &InvalidConfigError{Key: "start_date", Value: c.StartDate, Reason: "must be a date in YYYY-MM-DD form"})
	} else {
		start = t
	}

	if c.EndDate.String() == "" {
		errs = append(errs, &InvalidConfigError{Key: "end_date", Reason: "required"})
	} else if t, err := time.Parse(dateLayout, c.EndDate.String()); err != nil {
		errs = append(errs, &InvalidConfigError{Key: "end_date", Value: c.EndDate, Reason: "must be a date in YYYY-MM-DD form"})
	} else {
		end = t
	}

	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		errs = append(errs, &InvalidConfigError{Key: "end_date", Value: c.EndDate, Reason: "end date cannot be before start date"})
	}

	if c.Map.Width < minMapSide || c.Map.Width > maxMapSide {
		errs = append(errs, &InvalidConfigError{
			Key:    "map.width",
			Value:  c.Map.Width,
			Reason: fmt.Sprintf("must be between %d and %d", minMapSide, maxMapSide),
		})
	}
	if c.Map.Height < minMapSide || c.Map.Height > maxMapSide {
		errs = append(errs, &InvalidConfigError{
			Key:    "map.height",
			Value:  c.Map.Height,
			Reason: fmt.Sprintf("must be between %d and %d", minMapSide, maxMapSide),
		})
	}
	if c.Map.MaxZoom > maxZoomCeiling {
		errs = append(errs, &InvalidConfigError{
			Key:    "map.max_zoom",
			Value:  c.Map.MaxZoom,
			Reason: fmt.Sprintf("must be at most %d", maxZoomCeiling),
		})
	}

	if c.Privacy.CoordinatePrecision < 0 || c.Privacy.CoordinatePrecision > maxCoordinatePrecision {
		errs = append(errs, &InvalidConfigError{
			Key:    "privacy.coordinate_precision",
			Value:  c.Privacy.CoordinatePrecision,
			Reason: fmt.Sprintf("must be between 0 and %d", maxCoordinatePrecision),
		})
	}
	// Indexed by position, not by label: labels are optional and may repeat, so
	// the index is the only handle that always points the reader at one entry.
	for i, z := range c.Privacy.ExclusionZones {
		if z.RadiusM <= 0 {
			errs = append(errs, &InvalidConfigError{
				Key:    fmt.Sprintf("privacy.exclusion_zones[%d].radius_m", i),
				Value:  z.RadiusM,
				Reason: "must be positive",
			})
		}
	}

	// Case-sensitive, like every other YAML scalar: accepting GPS_then_time
	// would invent a spelling PRD §6.5 does not define.
	switch c.Photos.MatchMode {
	case "gps", "time", "gps_then_time":
	default:
		// An emptied key echoes nothing: "invalid value :" would leave two colons
		// with no input between them. Any other value is quoted.
		e := &InvalidConfigError{Key: "photos.match_mode", Reason: "must be one of " + matchModes}
		if c.Photos.MatchMode != "" {
			e.Value = c.Photos.MatchMode
		}
		errs = append(errs, e)
	}

	return errs.Err()
}
