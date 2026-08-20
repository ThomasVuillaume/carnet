package gpx

import "testing"

// TestBounds asserts exactly, at no tolerance. Bounds performs no arithmetic:
// every value it returns is a coordinate copied verbatim out of some point, so
// the expectations below are literals rather than expressions, and no platform
// can move them by an ULP.
//
// Two cases carry most of the weight. "empty track" pins the boolean, without
// which an absent extent would reach zoom selection disguised as a rectangle
// over the Gulf of Guinea. "southern and western hemispheres" holds no
// coordinate above zero, so an implementation seeding its minima and maxima at
// zero instead of at the first point returns 0 for both maxima and stays green
// on every other case in this table.
func TestBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		track      Track
		wantBounds Bounds
		wantOK     bool
	}{
		{
			name:       "empty track",
			track:      Track{},
			wantBounds: Bounds{},
			wantOK:     false,
		},
		{
			name:       "one empty segment",
			track:      Track{Segments: []Segment{{Points: []Point{}}}},
			wantBounds: Bounds{},
			wantOK:     false,
		},
		{
			name: "single point, degenerate rectangle",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 45.1204, Lon: 5.4331},
				}},
			}},
			wantBounds: Bounds{MinLat: 45.1204, MinLon: 5.4331, MaxLat: 45.1204, MaxLon: 5.4331},
			wantOK:     true,
		},
		{
			name: "one segment, extremes on different points",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 44.6821, Lon: 5.7433},
					{Lat: 45.2019, Lon: 5.1000},
					{Lat: 44.9000, Lon: 4.9012},
				}},
			}},
			wantBounds: Bounds{MinLat: 44.6821, MinLon: 4.9012, MaxLat: 45.2019, MaxLon: 5.7433},
			wantOK:     true,
		},
		{
			name: "two segments, each holding one extreme",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: 45.2019, Lon: 5.7433},
				}},
				{Points: []Point{
					{Lat: 44.6821, Lon: 4.9012},
				}},
			}},
			wantBounds: Bounds{MinLat: 44.6821, MinLon: 4.9012, MaxLat: 45.2019, MaxLon: 5.7433},
			wantOK:     true,
		},
		{
			name: "empty segment between two populated ones",
			track: Track{Segments: []Segment{
				{Points: []Point{{Lat: 45.2019, Lon: 5.7433}}},
				{Points: []Point{}},
				{Points: []Point{{Lat: 44.6821, Lon: 4.9012}}},
			}},
			wantBounds: Bounds{MinLat: 44.6821, MinLon: 4.9012, MaxLat: 45.2019, MaxLon: 5.7433},
			wantOK:     true,
		},
		{
			name: "leading segment empty",
			track: Track{Segments: []Segment{
				{Points: []Point{}},
				{Points: []Point{
					{Lat: 45.2019, Lon: 5.7433},
					{Lat: 44.6821, Lon: 4.9012},
				}},
			}},
			wantBounds: Bounds{MinLat: 44.6821, MinLon: 4.9012, MaxLat: 45.2019, MaxLon: 5.7433},
			wantOK:     true,
		},
		{
			name: "southern and western hemispheres, no coordinate above zero",
			track: Track{Segments: []Segment{
				{Points: []Point{
					{Lat: -33.4489, Lon: -70.6693},
					{Lat: -34.6037, Lon: -58.3816},
				}},
			}},
			wantBounds: Bounds{MinLat: -34.6037, MinLon: -70.6693, MaxLat: -33.4489, MaxLon: -58.3816},
			wantOK:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tc.track.Bounds()
			if ok != tc.wantOK {
				t.Fatalf("Bounds() ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.wantBounds {
				t.Errorf("Bounds() = %+v, want %+v", got, tc.wantBounds)
			}
		})
	}
}
