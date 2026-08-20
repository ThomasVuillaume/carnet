package geo

import "math"

// MaxLatitude is the pole-most latitude Web Mercator can represent, in degrees.
//
// It is atan(sinh(pi)) expressed in degrees, and it is not a rounded-off
// convention: it is the latitude where the projection reaches the edge of its
// own square. Beyond it y leaves [0, 1] and diverges to infinity at the poles,
// which is why the tile pyramid stops there and why no slippy map on the web
// shows the ice caps.
//
// Written as a literal rather than computed at init: this is a value from the
// specification, and a reader comparing carnet against EPSG:3857 must find the
// same digits here as in the standard.
const MaxLatitude = 85.0511287798066

// Project converts a WGS84 coordinate to normalised Web Mercator (EPSG:3857),
// returning x and y in [0, 1] — the whole world in a unit square, with (0, 0)
// at its north-west corner.
//
// Zoom and tile size are deliberately absent. Callers scale the result
// themselves with x * 2^z * TileSize, which keeps the transcendental part of
// the projection out of loops that try several zoom levels on the same
// coordinate.
//
// y grows southwards, opposite to latitude. The maximum latitude of a bounding
// box therefore yields its minimum y, and any caller computing a height must
// subtract in that order.
//
// Latitude is clamped to ±MaxLatitude; longitude is not clamped at all. The
// asymmetry is deliberate. A latitude past the square has no representation to
// fall back on, and the edge is the closest true statement the projection can
// make about it. A longitude past ±180° does have one — wrap it round the
// globe — but wrapping is a decision about the antimeridian that Bounds has
// already declined to take, and taking it here alone would hide the
// inconsistency rather than resolve it. Such a longitude therefore yields an x
// outside [0, 1], where it stays visible.
func Project(lat, lon float64) (x, y float64) {
	// Beyond the square Mercator can represent, the projection has no finite
	// answer, so callers are given its edge rather than an infinity. Clamping
	// here rather than returning an error is a judgement about the input
	// carnet actually sees: a motorcycle GPX recorded past 85° would be a
	// corrupt file, not a ride, and every caller of a projection is a loop
	// over thousands of points that has nowhere sensible to put an error.
	lat = min(max(lat, -MaxLatitude), MaxLatitude)

	// Longitude is linear in Mercator — the meridians are evenly spaced — so
	// this needs no radians and no transcendental function. It is also exact
	// at both ends: -180° gives 0 and 180° gives 1, with no rounding to
	// absorb due to Pi.
	x = (lon + 180) / 360

	// ln(tan(π/4 + φ/2)); atanh(sin φ), asinh(tan φ) and ln(tan φ + sec φ) are
	// the same function and agree to within an ULP over the clamped range.
	// This form is the steadiest of the four near the edge, where it evaluates
	// a tangent of 87.5° rather than summing two quantities that both diverge.
	//
	// The result spans [-π, π], counted positive northwards.
	// Formulas :
	// - OpenStreet Map : https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames
	// - IGN : https://data.geopf.fr/annexes/ressources/documentation/geodesie/algorithmes/alg0076.pdf
	// - Proj : https://proj.org/en/stable/operations/projections/webmerc.html
	const degToRad = math.Pi / 180
	gdInv := math.Log(math.Tan(math.Pi/4 + lat*degToRad/2))

	// Flip and rescale in one step: north goes from +π to 0, south from -π
	// to 1. The Earth's radius never enters the computation — EPSG:3857 would
	// multiply by 6378137 m to obtain metres, and dividing by the 2πR
	// circumference to normalise cancels it algebraically. Note that this
	// radius is the WGS84 semi-major axis, a different quantity from the mean
	// radius earthRadiusM that DistanceM uses; keeping both out of this
	// function keeps them from being confused for one another.
	y = (1 - gdInv/math.Pi) / 2

	// Restore the range the signature promises. At exactly ±MaxLatitude the
	// expression above lands 1.11e-16 the wrong side of the boundary: the
	// constant is a decimal literal, so it does not sit precisely where
	// gd⁻¹ equals π, and the residue escapes the interval.
	//
	// This corrects rounding on a value already known to be in range, unlike
	// the clamp on lat above, which corrects an input that genuinely was not.
	// It matters downstream rather than here: internal/tiles will turn y into
	// a tile index by flooring it, and a negative sliver floors to -1 — a tile
	// that does not exist, from a coordinate that was never out of bounds.
	y = min(max(y, 0), 1)

	return x, y
}
