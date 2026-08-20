package gpx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// SyntaxError reports a GPX file that could not be read as XML at all, as
// opposed to one that is well-formed but holds unusable data. Callers tell a
// corrupt download from a merely empty track with errors.As.
type SyntaxError struct {
	Err error
}

func (e *SyntaxError) Error() string { return "gpx: malformed XML: " + e.Err.Error() }
func (e *SyntaxError) Unwrap() error { return e.Err }

// errSkipPoint tells Parse to drop one point and carry on rather than reject
// the whole file. Wrap it with %w to keep a reason attached.
var errSkipPoint = errors.New("skip point")

// gpxFile mirrors the subset of GPX that carnet consumes. Element names carry
// no namespace prefix on purpose: encoding/xml then matches on local name
// alone, so GPX 1.0 files parse as well as 1.1. Unknown children, such as the
// Garmin gpxtpx:TrackPointExtension blocks in the fixture, are ignored.
type gpxFile struct {
	XMLName xml.Name `xml:"gpx"`
	Tracks  []xmlTrk `xml:"trk"`
}

type xmlTrk struct {
	Name     string      `xml:"name"`
	Segments []xmlTrkseg `xml:"trkseg"`
}

type xmlTrkseg struct {
	Points []xmlTrkpt `xml:"trkpt"`
}

// xmlTrkpt keeps Time as a string so that a malformed timestamp is pointFrom's
// decision rather than a decoding failure that loses the whole file.
type xmlTrkpt struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
}

// pointFrom converts one raw <trkpt> into a Point, and owns the parser's
// tolerance policy: which defects fail the whole file, and which disqualify a
// single point.
//
// A missing <time> is not a defect — hand-drawn routes and itinerary exports
// carry none, and their geometry is still worth mapping — so such a point keeps
// its coordinates and leaves Point.Time zero. A <time> present but unreadable
// is a defect: corrupt data says something absent data does not.
func pointFrom(raw xmlTrkpt) (Point, error) {
	// NaN must be rejected before the range checks: every comparison involving
	// NaN is false, so it would sail through both of them. strconv.ParseFloat,
	// which encoding/xml calls, accepts the literal "NaN".
	if math.IsNaN(raw.Lat) || math.IsNaN(raw.Lon) {
		return Point{}, fmt.Errorf("%w: coordinates are not numbers", errSkipPoint)
	}
	if raw.Lat == 0 && raw.Lon == 0 {
		// A real position in the Gulf of Guinea, but far more likely a <trkpt>
		// with no lat/lon attributes: encoding/xml leaves those at zero and
		// reports nothing.
		return Point{}, fmt.Errorf("%w: lat/lon both zero", errSkipPoint)
	}
	if raw.Lat < -90 || raw.Lat > 90 {
		return Point{}, fmt.Errorf("%w: lat %f out of range", errSkipPoint, raw.Lat)
	}
	if raw.Lon < -180 || raw.Lon > 180 {
		return Point{}, fmt.Errorf("%w: lon %f out of range", errSkipPoint, raw.Lon)
	}

	pt := Point{Lat: raw.Lat, Lon: raw.Lon}
	if raw.Time == "" {
		return pt, nil
	}

	t, err := time.Parse(time.RFC3339, raw.Time)
	if err != nil {
		return Point{}, fmt.Errorf("%w: timestamp %q: %w", errSkipPoint, raw.Time, err)
	}
	pt.Time = t

	return pt, nil
}

// Parse reads a GPX document and returns its first track. Empty segments are
// dropped; a document with no usable point yields a Track with no segment
// rather than an error, leaving that judgement to the caller.
func Parse(r io.Reader) (*Track, error) {
	var doc gpxFile
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, &SyntaxError{Err: err}
	}
	if len(doc.Tracks) == 0 {
		return &Track{}, nil
	}

	src := doc.Tracks[0]
	track := &Track{Name: src.Name}

	for segIdx, seg := range src.Segments {
		points := make([]Point, 0, len(seg.Points))
		for ptIdx, raw := range seg.Points {
			pt, err := pointFrom(raw)
			if errors.Is(err, errSkipPoint) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("gpx: segment %d, point %d: %w", segIdx, ptIdx, err)
			}
			points = append(points, pt)
		}
		if len(points) > 0 {
			track.Segments = append(track.Segments, Segment{Points: points})
		}
	}

	return track, nil
}
