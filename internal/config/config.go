package config

// Config is the whole of carnet.yaml. Title, StartDate and Track are required;
// every other key has a default, which is why Load decodes into Default rather
// than into a zero value.
//
// Track and PhotosDir are relative to the trip directory, not to the working
// directory, and are resolved after loading.
type Config struct {
	Title     string   `yaml:"title"`
	StartDate string   `yaml:"start_date"`
	EndDate   string   `yaml:"end_date"`
	Summary   string   `yaml:"summary"`
	Track     string   `yaml:"track"`
	PhotosDir string   `yaml:"photos_dir"`
	Tags      []string `yaml:"tags"`
	Map       Map      `yaml:"map"`
	Privacy   Privacy  `yaml:"privacy"`
	Photos    Photos   `yaml:"photos"`
}

// Map governs the rendered PNG. Width, Height and Padding are in pixels, and
// padding costs twice per axis since it applies to both opposite edges.
// TrackColor is an "#rrggbb" string. MaxZoom caps the ladder zoom selection
// walks down; the level actually used is the largest one the track fits in.
type Map struct {
	Width      int    `yaml:"width"`
	Height     int    `yaml:"height"`
	MaxZoom    int    `yaml:"max_zoom"`
	Padding    int    `yaml:"padding"`
	TrackColor string `yaml:"track_color"`
	TrackWidth int    `yaml:"track_width"`
	TileSource string `yaml:"tile_source"`
}

// Privacy drives the pipeline of PRD §6.6, whose order is fixed: exclusion
// zones, then trimming, then coordinate rounding, then EXIF purge.
type Privacy struct {
	ExclusionZones []ExclusionZone `yaml:"exclusion_zones"`

	// Metres cut from each end of every segment. Zero disables trimming and is
	// not the same answer as an absent key, which restores the default.
	TrimStartM int `yaml:"trim_start_m"`
	TrimEndM   int `yaml:"trim_end_m"`

	// Decimals kept on published coordinates. Three of them leave about 110 m
	// of uncertainty.
	CoordinatePrecision int `yaml:"coordinate_precision"`

	StripExif bool `yaml:"strip_exif"`

	// GPS tags are never kept, whatever this lists. Hard-coded rule.
	KeepExifTags []string `yaml:"keep_exif_tags"`
}

// ExclusionZone is a circle nothing published may fall inside. Centre in
// decimal degrees (WGS84), radius in metres. Track points inside are deleted,
// splitting the segment where the track crosses it, and a photo resolved
// inside leaves the output entirely rather than losing its position. Label
// names the zone in warnings.
type ExclusionZone struct {
	Lat     float64 `yaml:"lat"`
	Lon     float64 `yaml:"lon"`
	RadiusM int     `yaml:"radius_m"`
	Label   string  `yaml:"label"`
}

// Photos governs photo/track correlation. ClockOffsetS is added to EXIF
// timestamps before matching, to correct a camera clock. MatchMode is "gps",
// "time" or "gps_then_time". When both methods answer and disagree by more
// than DivergenceWarnM, the GPS position wins and a warning names the photo.
type Photos struct {
	ClockOffsetS    int    `yaml:"clock_offset_s"`
	MatchToleranceS int    `yaml:"match_tolerance_s"`
	MatchMode       string `yaml:"match_mode"`
	DivergenceWarnM int    `yaml:"divergence_warn_m"`
	CaptionsFile    string `yaml:"captions_file"`
}
