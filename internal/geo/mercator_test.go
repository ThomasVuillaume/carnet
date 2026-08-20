package geo

import (
	"math"
	"testing"
)

// TestMaxLatitude checks the literal against its definition, atan(sinh(pi)) in
// degrees. A typo in its last places would go unnoticed otherwise, and every
// boundary case in TestProject would follow it.
//
// The tolerance covers the literal's 16 significant digits, plus platform
// differences in Atan and Sinh.
func TestMaxLatitude(t *testing.T) {
	t.Parallel()

	const tolerance = 1e-12

	want := math.Atan(math.Sinh(math.Pi)) * 180 / math.Pi
	if math.Abs(MaxLatitude-want) > tolerance {
		t.Errorf("MaxLatitude = %v, want %v (tolerance %g)", MaxLatitude, want, tolerance)
	}
}

// TestProject keeps the tolerance per case, as TestDistanceM does. Three
// regimes: x is affine and bit-exact; y goes through Tan and Log, which the
// standard does not require to be correctly rounded, so mid-square cases carry
// a few ULP; edge cases are exact again because the clamp returns the bound
// itself rather than approaching it.
//
// Expectations were computed outside this package through atanh(sin(phi)), a
// different form of the inverse Gudermannian than the implementation uses, so
// the table does not replay our own arithmetic back at us.
//
// Two properties are asserted on every row rather than as cases of their own:
// hemispheric symmetry, y(phi) + y(-phi) = 1 since gd⁻¹ is odd, which catches a
// sign error in the flip; and separability, x depending on lon alone and y on
// lat alone, asserted with == because both calls run identical arithmetic.
//
// The domain guard covers y only: Project bounds it unconditionally but
// promises nothing for x outside lon in [-180, 180], which the
// "longitude past the antimeridian" case pins.
//
// The NaN guard carries weight because math.Abs(NaN-want) > tol is false — a
// NaN would satisfy every value assertion in silence. It is what makes the
// beyond-the-pole case bite, and that case is the only one that fails if the
// clamp on lat is removed: at 89 or 90 degrees the clamp on y repairs the
// result and hides the loss. Checked by mutation.
func TestProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lat, lon  float64
		wantX     float64
		wantY     float64
		tolerance float64 // absolute, on normalised coordinates
	}{
		{
			name: "origin",
			lat:  0, lon: 0,
			wantX:     0.5,
			wantY:     0.5,
			tolerance: 0,
		},
		{
			name: "antimeridian, western edge",
			lat:  0, lon: -180,
			wantX:     0,
			wantY:     0.5,
			tolerance: 0,
		},
		{
			name: "antimeridian, eastern edge",
			lat:  0, lon: 180,
			wantX:     1,
			wantY:     0.5,
			tolerance: 0,
		},
		{
			// Pinned so that adding a wrap later is a deliberate change to a
			// stated contract rather than a silent one.
			name: "longitude past the antimeridian is not wrapped",
			lat:  0, lon: 200,
			wantX:     1.0555555555555556,
			wantY:     0.5,
			tolerance: 0,
		},
		{
			name: "Vercors foothills",
			lat:  45.1885, lon: 5.7245,
			wantX:     0.5159013888888889,
			wantY:     0.35898331686188,
			tolerance: 1e-12,
		},
		{
			// Both coordinates negative: neither half of the result may come
			// back from the northern or eastern side by accident.
			name: "southern and western hemispheres",
			lat:  -33.4489, lon: -70.6693,
			wantX:     0.3036963888888889,
			wantY:     0.5986909293552369,
			tolerance: 1e-12,
		},
		{
			// gd⁻¹ reaches pi here, so y is 0 — but only after the clamp, the
			// raw expression landing 1.11e-16 below zero. At zero tolerance
			// this case fails if that clamp goes away.
			name: "northern edge of the square",
			lat:  MaxLatitude, lon: 0,
			wantX:     0.5,
			wantY:     0,
			tolerance: 0,
		},
		{
			name: "southern edge of the square",
			lat:  -MaxLatitude, lon: 0,
			wantX:     0.5,
			wantY:     1,
			tolerance: 0,
		},
		{
			name: "latitude past the northern edge",
			lat:  89, lon: 0,
			wantX:     0.5,
			wantY:     0,
			tolerance: 0,
		},
		{
			name: "latitude past the southern edge",
			lat:  -89, lon: 0,
			wantX:     0.5,
			wantY:     1,
			tolerance: 0,
		},
		{
			// Past the pole the tangent changes sign and Log returns NaN, which
			// the clamp on y cannot repair. Only the clamp on lat keeps this
			// finite, which makes it the one row that tests it.
			name: "latitude past the pole",
			lat:  91, lon: 0,
			wantX:     0.5,
			wantY:     0,
			tolerance: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			x, y := Project(tc.lat, tc.lon)

			if math.IsNaN(x) || math.IsNaN(y) {
				t.Fatalf("Project(%v, %v) = (%v, %v), want (%v, %v)", tc.lat, tc.lon, x, y, tc.wantX, tc.wantY)
			}
			if y < 0 || y > 1 {
				t.Errorf("Project(%v, %v) y = %v, outside [0, 1]", tc.lat, tc.lon, y)
			}

			if math.Abs(x-tc.wantX) > tc.tolerance {
				t.Errorf("Project(%v, %v) x = %v, want %v (tolerance %g)", tc.lat, tc.lon, x, tc.wantX, tc.tolerance)
			}
			if math.Abs(y-tc.wantY) > tc.tolerance {
				t.Errorf("Project(%v, %v) y = %v, want %v (tolerance %g)", tc.lat, tc.lon, y, tc.wantY, tc.tolerance)
			}

			const symmetryTolerance = 1e-15

			_, mirrored := Project(-tc.lat, tc.lon)
			if math.Abs(y+mirrored-1) > symmetryTolerance {
				t.Errorf("y(%v) + y(%v) = %v, want 1 (tolerance %g)", tc.lat, -tc.lat, y+mirrored, symmetryTolerance)
			}

			if xAtEquator, _ := Project(0, tc.lon); xAtEquator != x {
				t.Errorf("x depends on lat: Project(%v, %v) x = %v, Project(0, %v) x = %v", tc.lat, tc.lon, x, tc.lon, xAtEquator)
			}
			if _, yAtGreenwich := Project(tc.lat, 0); yAtGreenwich != y {
				t.Errorf("y depends on lon: Project(%v, %v) y = %v, Project(%v, 0) y = %v", tc.lat, tc.lon, y, tc.lat, yAtGreenwich)
			}
		})
	}
}
