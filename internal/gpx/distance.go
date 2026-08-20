package gpx

import "carnet/internal/geo"

// LengthM returns the distance ridden along the track, in metres: the sum of
// its segments' lengths.
//
// Gaps between segments are not bridged. A segment boundary means the recording
// stopped, or a privacy exclusion zone cut the track in two, and neither is
// ground the rider covered.
//
// The value is not rounded. internal/output decides the decimals when it writes
// distance_km; internal/privacy needs full precision for trimming and
// exclusion-zone tests.
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
// The order is load-bearing. Floating point addition is not associative, so
// summing the same distances differently changes the last bits, and carnet
// requires byte-identical output across runs. Recording order is fixed by the
// file — which is also what rules out parallelising the loop.
//
// A segment of one point or none has no consecutive pair and measures zero.
// The loop bounds already say so; no guard repeats it.
func (s Segment) LengthM() float64 {
	var total float64
	for i := 1; i < len(s.Points); i++ {
		total += geo.DistanceM(s.Points[i-1].Lat, s.Points[i-1].Lon, s.Points[i].Lat, s.Points[i].Lon)
	}

	return total
}
