package gpx

import "carnet/internal/geo"

// LengthM returns the distance ridden along the track, in metres: the sum of
// its segments' lengths.
//
// Gaps between segments are not bridged. A segment boundary means the
// recording stopped, or a privacy exclusion zone cut the track in two, and
// neither is ground the rider covered.
//
// The value is not rounded. Rounding belongs to whoever publishes it —
// internal/output writes distance_km to the frontmatter and decides on the
// decimals there — while internal/privacy needs the full precision for
// trimming and exclusion-zone tests.
func (t *Track) LengthM() float64 {
	var total float64
	for _, seg := range t.Segments {
		total += seg.LengthM()
	}

	return total
}

// LengthM returns the length of the segment, in metres: the great-circle
// distance between each consecutive pair of points, summed in recording order.
//
// The order matters beyond readability. Floating point addition is not
// associative, so summing the same distances in another order changes the last
// bits of the result, and carnet requires byte-identical output across runs.
// Recording order is fixed by the file, which is what makes this reproducible
// — and what rules out parallelising the loop.
//
// A segment holding one point, or none, has no consecutive pair and measures
// zero. The loop bounds already state that; no guard is needed to say it
// twice.
func (s Segment) LengthM() float64 {
	var total float64
	for i := 1; i < len(s.Points); i++ {
		total += geo.DistanceM(s.Points[i-1].Lat, s.Points[i-1].Lon, s.Points[i].Lat, s.Points[i].Lon)
	}

	return total
}
