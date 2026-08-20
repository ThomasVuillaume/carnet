package gpx

import "time"

// Point is a single recorded position, in decimal degrees (WGS84).
//
// Time is optional: some devices omit <time>. Callers must test it with
// IsZero rather than comparing against a literal, and packages that need a
// timestamp — photo correlation, moving time — must tolerate its absence.
type Point struct {
	Lat  float64
	Lon  float64
	Time time.Time
}

// Segment is a contiguous run of points recorded without interruption. A GPX
// track may contain several, and privacy exclusion zones can split one segment
// into several more.
type Segment struct {
	Points []Point
}

// Track is a complete recorded activity: an optional name and its segments, in
// file order.
type Track struct {
	Name     string
	Segments []Segment
}

// Bounds is the geographic extent of a track, in decimal degrees.
type Bounds struct {
	MinLat, MinLon float64
	MaxLat, MaxLon float64
}

// firstPoint reports whether the track contains any point at all.
// It returns the first point found, or the zero value if none exists.
func (t *Track) firstPoint() (Point, bool) {
	for _, segment := range t.Segments {
		if len(segment.Points) > 0 {
			return segment.Points[0], true
		}
	}
	return Point{}, false
}
