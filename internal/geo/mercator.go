package geo

import "math"

// MaxLatitude is the pole-most latitude Web Mercator can represent, in degrees:
// atan(sinh(pi)), where the projection reaches the edge of its square. Beyond
// it, y leaves [0, 1] and diverges at the poles.
//
// A literal rather than a computed value, so that it can be matched against the
// EPSG:3857 specification digit for digit.
const MaxLatitude = 85.0511287798066

// Project converts a WGS84 coordinate to normalised Web Mercator (EPSG:3857):
// x and y in [0, 1], the whole world in a unit square with (0, 0) at its
// north-west corner.
//
// Zoom and tile size are left to callers, who scale with x * 2^z * TileSize.
// This keeps the transcendental part out of loops that try several zoom levels
// on one coordinate.
//
// y grows southwards. A bounding box's maximum latitude yields its minimum y,
// so callers computing a height must subtract in that order.
//
// Latitude is clamped to ±MaxLatitude, longitude is not. A latitude past the
// square has no representation to fall back on; a longitude past ±180° has one
// — wrapping — but that is an antimeridian decision Bounds has already
// declined, and taking it here alone would hide the inconsistency. Such a
// longitude yields an x outside [0, 1].
//
// References:
//   - OpenStreetMap: https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames
//   - IGN: https://data.geopf.fr/annexes/ressources/documentation/geodesie/algorithmes/alg0076.pdf
//   - PROJ: https://proj.org/en/stable/operations/projections/webmerc.html
func Project(lat, lon float64) (x, y float64) {
	// Past the square the projection has no finite value, so callers get its
	// edge rather than an infinity. An error would be truer but unusable: a
	// GPX recorded past 85° is a corrupt file, and every caller is a loop over
	// thousands of points with nowhere to put one.
	lat = min(max(lat, -MaxLatitude), MaxLatitude)

	// Meridians are evenly spaced, so x needs no radians and no transcendental.
	// Exact at both ends: -180° gives 0, 180° gives 1.
	x = (lon + 180) / 360

	// The inverse Gudermannian, spanning [-π, π] positive northwards.
	// atanh(sin φ), asinh(tan φ) and ln(tan φ + sec φ) are the same function;
	// this form is the steadiest near the edge, evaluating a tangent of 87.5°
	// rather than summing two diverging quantities.
	const degToRad = math.Pi / 180
	gdInv := math.Log(math.Tan(math.Pi/4 + lat*degToRad/2))

	// Flip and rescale: north from +π to 0, south from -π to 1. The Earth's
	// radius cancels — EPSG:3857 multiplies by 6378137 m, normalising divides
	// by the 2πR circumference. That radius is the WGS84 semi-major axis, not
	// the mean radius earthRadiusM that DistanceM uses.
	y = (1 - gdInv/math.Pi) / 2

	// At exactly ±MaxLatitude the expression above lands 1.11e-16 outside the
	// promised range: the constant is a decimal literal and does not sit
	// precisely where gd⁻¹ equals π. Unlike the clamp on lat, this corrects
	// rounding on a value already in range. It matters downstream, where
	// internal/tiles floors y into a tile index and a negative sliver floors
	// to -1.
	y = min(max(y, 0), 1)

	return x, y
}
