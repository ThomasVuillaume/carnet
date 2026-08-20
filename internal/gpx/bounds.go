package gpx

import "carnet/internal/geo"

// Bounds returns the smallest lat/lon rectangle enclosing every point of every
// segment, and reports whether the track held any point at all. When it did
// not, the returned Bounds is the zero value and must not be read.
//
// The boolean is not decoration. The zero Bounds is a perfectly valid
// rectangle, collapsed on 0°N 0°E in the Gulf of Guinea — the very coordinates
// pointFrom already treats as the tell of a <trkpt> missing its attributes. A
// track with no point has no extent, and saying so out of band is what keeps
// an absent answer from being read as a real one downstream, where zoom
// selection would happily frame a map on the Atlantic.
//
// Longitude is compared as a plain number, so a track crossing the
// antimeridian yields a rectangle spanning the wrong way round the globe. That
// limit is accepted, not overlooked: carnet maps rides that never come near
// ±180°, and handling the general case would force a wrap-around flag on every
// caller for a case none of them will meet.
//
// Callers that publish the result must reorder it. The frontmatter bbox is
// [minLon, minLat, maxLon, maxLat] — GeoJSON order, longitude first — which is
// not the field order of this struct.
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
