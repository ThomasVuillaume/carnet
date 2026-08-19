package gpx

import (
	"math"
	"testing"

	"carnet/internal/geo"
)

// TestLengthM exercises the summation, not the distance formula:
// haversine_test.go already pins geo.DistanceM. That is why most expectations
// are built from geo.DistanceM itself. Such a case would stay green if the
// formula were wrong, but turns red the moment LengthM pairs the wrong points,
// drops a segment, or bridges the gap between two — which is what this test is
// for.
//
// One case steps outside that circularity on purpose. "one segment, two
// points" carries a literal 111195.08 m, one degree of longitude at the
// equator, so that an error of unit — a stray division by 1000 — has somewhere
// to fail. Its tolerance is half a metre: loose enough that Sin, Cos and Asin
// may each land an ULP apart from one architecture to the next, tight enough
// that no unit mistake survives it.
//
// The other cases assert exactly, at tolerance zero, and can afford to. Adding
// zero to a float returns it unchanged, and doubling is exact in binary
// floating point, so the arithmetic the table performs is bit for bit the
// arithmetic LengthM performs. Nothing there depends on the platform.
func TestLengthM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		track      Track
		wantLength float64
		tolerance  float64 // absolute tolerance in metres
	}{
		{
			name:       "empty track",
			track:      Track{},
			wantLength: 0,
			tolerance:  0,
		},
		{
			name:       "empty segment",
			track:      Track{Segments: []Segment{{Points: []Point{}}}},
			wantLength: 0,
			tolerance:  0,
		},
		{
			name: "one segment, one point",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 0, Lon: 1},
				}},
			}},
			wantLength: 0,
			tolerance:  0,
		},
		{
			name: "one segment, two points",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 0, Lon: 0},
					{Lat: 0, Lon: 1},
				}},
			}},
			wantLength: 111_195.08, // one degree of longitude at the equator
			tolerance:  0.5,
		},
		{
			name: "two segments, two points each",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 0, Lon: 0},
					{Lat: 0, Lon: 1},
				}},
				{Points: []Point{
					{Lat: 1, Lon: 1},
					{Lat: 1, Lon: 2},
				}},
			}},
			wantLength: geo.DistanceM(0, 0, 0, 1) + geo.DistanceM(1, 1, 1, 2),
			tolerance:  0,
		},
		{
			name: "one segment, three points back-and-forth, double distance",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 0, Lon: 0},
					{Lat: 0, Lon: 1},
					{Lat: 0, Lon: 0},
				}},
			}},
			wantLength: geo.DistanceM(0, 0, 0, 1) * 2,
			tolerance:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.track.LengthM()
			if math.Abs(got-tc.wantLength) > tc.tolerance {
				t.Errorf("LengthM() = %v, want %v (tolerance %v)", got, tc.wantLength, tc.tolerance)
			}
		})
	}
}
