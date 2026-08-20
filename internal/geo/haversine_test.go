package geo

import (
	"math"
	"testing"
)

// TestDistanceM mixes expectations of three kinds, hence a per-case tolerance:
// one package-level epsilon would impose the loosest case on the strictest, and
// two of these are exact.
//
// Derived expectations, at no tolerance. Along a meridian from the equator the
// true distance reduces to earthRadiusM * phi2, which follows from our own
// constant and which DistanceM returns bit for bit. The antipodal case is exact
// for another reason: the clamp maps every near-antipodal pair onto a == 1,
// hence asin(1) and exactly pi * earthRadiusM.
//
// The 5 to 15 m meridian cases are derived the same way but carry a nanometre,
// since Sin and Asin may differ by an ULP across architectures. That is four
// orders of magnitude tighter than the 0.18 mm by which the spherical law of
// cosines misses at this scale, so they still discriminate between formulas —
// which a tolerance taken from a map reading would not.
//
// One measured expectation, asserted loosely. The Eiffel Tower to Capitole
// distance comes from cartes.gouv.fr, whose model is not ours: its tolerance
// absorbs the gap between an ellipsoid and our sphere, not a computation error.
// Tightening it would test the reference rather than the code.
//
// The antipodal coordinates were found by sweeping, not chosen. The obvious
// candidates — (0,0) to (0,180), or pole to pole — never trigger the bug, sin
// and cos being exact on those arguments. Only arbitrary latitudes push the
// intermediate "a" past 1 and, before the clamp, made asin return NaN. Removing
// that clamp must turn this table red; checked by mutation.
//
// Hence the NaN guard ahead of every comparison: math.Abs(NaN-want) > tol is
// false, so a NaN would silently satisfy the assertions below.
func TestDistanceM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want                   float64
		tolerance              float64 // absolute tolerance in metres
	}{
		{
			name: "same point",
			lat1: 44.472103, lon1: 3.858475,
			lat2: 44.472103, lon2: 3.858475,
			want:      0,
			tolerance: 0,
		},
		{
			name: "one degree of latitude at equator",
			lat1: 0, lon1: 0,
			lat2: 1, lon2: 0,
			want:      earthRadiusM * math.Pi / 180,
			tolerance: 0,
		},
		{
			name: "Eiffel Tower to Capitole",
			lat1: 48.858264, lon1: 2.294498,
			lat2: 43.604474, lon2: 1.444217,
			want:      587_833, // Measured on cartes.gouv.fr
			tolerance: 2,
		},
		{
			name: "antipodal points",
			lat1: -42.3639, lon1: -171,
			lat2: 42.3639, lon2: 9,
			want:      math.Pi * earthRadiusM,
			tolerance: 0,
		},
		// The scale a real track is made of: consecutive trackpoints sit 5 to
		// 15 m apart, and a whole ride is that distance summed a few thousand
		// times. Each latitude below is the arc of the given length seen from
		// the centre of the sphere, so the expected value is the metre count
		// itself. Nothing here is read off a map: the tolerance is a few ULP
		// of slack for platform differences in Sin and Asin, not the accuracy
		// of a source.
		{
			name: "5 m along a meridian",
			lat1: 0, lon1: 0,
			lat2: (5.0 / earthRadiusM) / (math.Pi / 180), lon2: 0,
			want:      5,
			tolerance: 1e-9,
		},
		{
			name: "10 m along a meridian",
			lat1: 0, lon1: 0,
			lat2: (10.0 / earthRadiusM) / (math.Pi / 180), lon2: 0,
			want:      10,
			tolerance: 1e-9,
		},
		{
			name: "15 m along a meridian",
			lat1: 0, lon1: 0,
			lat2: (15.0 / earthRadiusM) / (math.Pi / 180), lon2: 0,
			want:      15,
			tolerance: 1e-9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := DistanceM(tc.lat1, tc.lon1, tc.lat2, tc.lon2)
			got2 := DistanceM(tc.lat2, tc.lon2, tc.lat1, tc.lon1)

			if math.IsNaN(got) || math.IsNaN(got2) {
				t.Fatalf("DistanceM(%v, %v, %v, %v) = NaN, want %v", tc.lat1, tc.lon1, tc.lat2, tc.lon2, tc.want)
			}

			if math.Abs(got2-got) > 1e-9 {
				t.Errorf("DistanceM is not symmetric: got %v, got2 %v", got, got2)
			}
			if math.Abs(got-tc.want) > tc.tolerance {
				t.Errorf("DistanceM(%v, %v, %v, %v) = %v, want %v", tc.lat1, tc.lon1, tc.lat2, tc.lon2, got, tc.want)
			}
		})
	}
}
