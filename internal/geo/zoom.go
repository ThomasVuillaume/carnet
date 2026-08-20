package geo

import "math"

// TileSize is the edge of a map tile, in pixels. Slippy map servers cut the
// world into 256 px squares — a convention of that protocol, not a property of
// the projection. The world is therefore TileSize * 2^z pixels wide at zoom z,
// which is what turns Project's [0, 1] fractions into pixels.
const TileSize = 256

// SelectZoom returns the largest zoom level at which b, inset by padding on
// every side, still fits in a width by height image. Dimensions are in pixels,
// and padding costs twice, once per opposite edge.
//
// Tile servers publish integer zoom levels and nothing between, so the fit is
// rounded down and leaves slack on at least one axis. Centring the box in that
// slack is the caller's job.
//
// A single point constrains nothing and yields maxZoom. A box that overflows
// even the whole-world view yields 0, as does a padding that leaves no room at
// all. Both ends of the ladder are answers, not error signals.
//
// b must come from a track that held at least one point. The zero Bounds is a
// valid rectangle collapsed on 0°N 0°E, so an empty track would be framed on
// the Atlantic rather than rejected. Track.Bounds reports emptiness out of band
// for that reason.
//
// The closed form, floor(log2(min(availWidth/dx, availHeight/dy))), divides by
// a dx that is zero on any north-south track: 0/0 yields NaN, and int(NaN) is
// architecture-dependent in Go, which breaks the byte-for-byte output promise.
// Fitting is monotone in z — true up to one level, false above — so scanning
// down from maxZoom reaches the same frontier with comparisons alone.
func SelectZoom(b Bounds, width, height, padding, maxZoom int) int {
	zoomSelected := 0

	availableWidth := width - 2*padding
	availableHeight := height - 2*padding

	westX, southY := Project(b.MinLat, b.MinLon)
	eastX, northY := Project(b.MaxLat, b.MaxLon)

	bboxWidth := eastX - westX

	// Project's y grows southwards, so MaxLat yields the smaller y and this
	// subtraction runs the opposite way round from the width above. Reversed,
	// it gives a negative height that satisfies every comparison below, and the
	// vertical constraint drops out unnoticed.
	bboxHeight := southY - northY

	for z := maxZoom; z >= 0; z-- {
		bboxWidthAtZoom := math.Ldexp(1, z) * TileSize * bboxWidth
		bboxHeightAtZoom := math.Ldexp(1, z) * TileSize * bboxHeight

		if bboxWidthAtZoom <= float64(availableWidth) && bboxHeightAtZoom <= float64(availableHeight) {
			zoomSelected = z
			break
		}
	}

	return zoomSelected
}
