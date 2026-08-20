package geo

import (
	"math"
	"testing"
)

// TestMaxLatitude checks the constant against its own definition, atan(sinh(pi))
// in degrees, computed here rather than transcribed. The literal in mercator.go
// exists so that a reader can match carnet against the EPSG:3857 specification
// digit for digit; that readability is worth nothing if a typo in the last
// places went unnoticed, and every boundary case in TestProject would happily
// follow a wrong constant into the wrong place.
//
// The tolerance covers the decimal literal being 16 significant digits of a
// value the computation returns to full precision, plus platform latitude in
// Atan and Sinh.
func TestMaxLatitude(t *testing.T) {
	t.Parallel()

	const tolerance = 1e-12

	want := math.Atan(math.Sinh(math.Pi)) * 180 / math.Pi
	if math.Abs(MaxLatitude-want) > tolerance {
		t.Errorf("MaxLatitude = %v, want %v (tolerance %g)", MaxLatitude, want, tolerance)
	}
}

// TestProject keeps the tolerance as a per-case field, for the same reason
// TestDistanceM does: some expectations here are exact and others are not. The
// x half of the projection is affine — (lon + 180) / 360 on well-conditioned
// values — so it is bit-exact and asserted at zero tolerance. The y half goes
// through Tan and Log, which the standard does not require to be correctly
// rounded, so the cases that land mid-square carry a few ULP of slack. The
// cases that land on an edge are exact again, but for a third reason: the
// clamp in Project returns the bound itself, not a computed approach to it.
//
// Expectations were computed outside this package and through atanh(sin(phi)),
// a different form of the inverse Gudermannian than the ln(tan(pi/4 + phi/2))
// the implementation uses. The table therefore discriminates between a correct
// projection and a plausible-looking one, instead of replaying our own
// arithmetic back at us.
//
// Two properties are asserted on every row rather than as cases of their own,
// the way TestDistanceM asserts symmetry:
//
// Hemispheric symmetry, y(phi) + y(-phi) = 1, because gd⁻¹ is odd. It catches a
// sign error in the flip that a single-hemisphere expectation would not, and it
// holds bit-exactly on every latitude tried — the tolerance is there for
// platform differences in Tan and Log, not for the identity.
//
// Separability: x depends on lon alone and y on lat alone, which is a property
// of Mercator itself, not of this implementation. Asserted with == because both
// calls run the same arithmetic on the same argument; anything else means a
// coordinate leaked across.
//
// The domain guard covers y only. Project promises [0, 1] for y
// unconditionally, the clamp seeing to it, but promises nothing for x outside
// lon in [-180, 180] — the "longitude past the antimeridian" case below exists
// precisely to pin that documented refusal to wrap.
//
// The NaN guard is not decoration either. math.Abs(NaN-want) > tol is false, so
// a NaN would satisfy the value assertions in silence. It is what makes the
// beyond-the-pole case bite: 91 degrees puts pi/4 + phi/2 past a right angle,
// Tan turns negative, Log returns NaN, and Go's min and max propagate it. That
// case is the only one in the table that fails if the clamp on lat is removed —
// at 89 or even 90 degrees the clamp on y quietly repairs the result and hides
// the loss.
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
			// Project documents that it does not wrap longitude, leaving the
			// result outside the unit square where it stays visible. Pinned
			// here so that adding a wrap later is a deliberate change to a
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
			// Both coordinates negative, and neither y nor x may come back
			// from the northern or eastern half by accident.
			name: "southern and western hemispheres",
			lat:  -33.4489, lon: -70.6693,
			wantX:     0.3036963888888889,
			wantY:     0.5986909293552369,
			tolerance: 1e-12,
		},
		{
			// The square's own edge: gd⁻¹ reaches pi here, so y is 0 — but only
			// after the clamp, the raw expression landing 1.11e-16 below zero.
			// At zero tolerance this case fails if that clamp goes away.
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
			// case finite, which makes it the one row that tests it.
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
