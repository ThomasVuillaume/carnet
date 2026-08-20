package config

// Defaults returns a Config with every default value of the format.
// Load decodes the file into this value rather than into a zero one: an absent
// key keeps its default, an explicit key overwrites it even when it carries
// zero. That distinction matters on privacy keys, where 0 means "no trimming
// at all" and absence means 3000 m.
func Defaults() Config {
	return Config{
		Tags:      nil,
		PhotosDir: "photos",
		Map: Map{
			Width:      1280,
			Height:     720,
			MaxZoom:    15,
			Padding:    32,
			TrackColor: "#c1121f",
			TrackWidth: 4,
			TileSource: "osm",
		},
		Privacy: Privacy{
			ExclusionZones:      nil,
			TrimStartM:          3000,
			TrimEndM:            3000,
			CoordinatePrecision: 3,
			StripExif:           true,
			KeepExifTags:        []string{"Make", "Model", "LensModel", "FocalLength", "FNumber", "ExposureTime", "ISOSpeedRatings"},
		},
		Photos: Photos{
			ClockOffsetS:    0,
			MatchToleranceS: 120,
			MatchMode:       "gps_then_time",
			DivergenceWarnM: 100,
			CaptionsFile:    "captions.yaml",
		},
	}
}
