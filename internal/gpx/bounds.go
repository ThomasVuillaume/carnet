package gpx

import "carnet/internal/geo"

// Bounds returns the smallest lat/lon rectangle enclosing every point of every
// segment, and reports whether the track held any point at all. When it did
// not, the returned Bounds is the zero value and must not be read.
//
// The boolean matters because the zero Bounds is itself a valid rectangle,
// collapsed on 0°N 0°E — the coordinates pointFrom already treats as the tell
// of a <trkpt> missing its attributes. Reporting emptiness out of band keeps
// zoom selection from framing a map on the Atlantic.
//
// Longitude is compared as a plain number, so a track crossing the antimeridian
// yields a rectangle spanning the wrong way round the globe. That limit is
// accepted: carnet maps rides that never come near ±180°, and the general case
// would force a wrap-around flag on every caller.
//
// Callers that publish the result must reorder it: the frontmatter bbox is
// [minLon, minLat, maxLon, maxLat], GeoJSON order, longitude first.
func (t *Track) Bounds() (geo.Bounds, bool) {
	var point, ok = t.firstPoint()

	if !ok {
		return geo.Bounds{}, false
	}

	minLat := point.Lat
	minLon := point.Lon
	maxLat := point.Lat
	maxLon := point.Lon

	for _, segment := range t.Segments {
		for _, pt := range segment.Points {
			if pt.Lat < minLat {
				minLat = pt.Lat
			}
			if pt.Lon < minLon {
				minLon = pt.Lon
			}
			if pt.Lat > maxLat {
				maxLat = pt.Lat
			}
			if pt.Lon > maxLon {
				maxLon = pt.Lon
			}
		}
	}

	return geo.Bounds{MinLat: minLat, MinLon: minLon, MaxLat: maxLat, MaxLon: maxLon}, true
}
