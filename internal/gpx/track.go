package gpx

import "time"

// Point is a single recorded position, in decimal degrees (WGS84).
//
// Time is optional: some devices omit <time>. Test it with IsZero rather than
// against a literal, and expect its absence wherever a timestamp is needed —
// photo correlation, moving time.
type Point struct {
	Lat  float64
	Lon  float64
	Time time.Time
}

// Segment is a contiguous run of points recorded without interruption. A GPX
// track may hold several, and privacy exclusion zones can split one into more.
type Segment struct {
	Points []Point
}

// Track is a complete recorded activity: an optional name and its segments, in
// file order.
type Track struct {
	Name     string
	Segments []Segment
}

// firstPoint returns the first point of the track, or false if it holds none.
func (t *Track) firstPoint() (Point, bool) {
	for _, segment := range t.Segments {
		if len(segment.Points) > 0 {
			return segment.Points[0], true
		}
	}
	return Point{}, false
}
