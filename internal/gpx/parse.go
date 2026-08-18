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
// opposed to one that is well-formed but holds unusable data. Callers that
// must tell a corrupt download from a merely empty track distinguish the two
// with errors.As.
type SyntaxError struct {
	Err error
}

func (e *SyntaxError) Error() string { return "gpx: malformed XML: " + e.Err.Error() }
func (e *SyntaxError) Unwrap() error { return e.Err }

// errSkipPoint, returned by pointFrom, tells Parse to drop one point and carry
// on rather than reject the whole file. Wrap it with %w to keep a reason
// attached.
var errSkipPoint = errors.New("skip point")

// gpxFile mirrors the subset of GPX that carnet consumes. Element names carry
// no namespace prefix on purpose: encoding/xml then matches on local name
// alone, so files declaring the GPX 1.0 namespace parse just as well as 1.1.
// Unknown children — the Garmin gpxtpx:TrackPointExtension blocks in the test
// fixture, for instance — are silently ignored by the decoder.
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

// xmlTrkpt keeps Time as a string so that a malformed timestamp is a decision
// for pointFrom rather than a decoding failure that loses the whole file.
type xmlTrkpt struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
}

// pointFrom converts one raw <trkpt> into a Point, and owns the parser's
// tolerance policy: which defects are worth failing the whole file over, and
// which merely disqualify a single point.
//
// A missing <time> is not a defect. Hand-drawn routes and itinerary exports
// carry none, and their geometry is still worth mapping; such a point keeps
// its coordinates and leaves Point.Time at its zero value. A <time> that is
// present but unreadable is treated as a defect, because corrupt data says
// something that absent data does not.
func pointFrom(raw xmlTrkpt) (Point, error) {
	// NaN has to be rejected before the range checks below. Every comparison
	// involving NaN is false, so both NaN < -90 and NaN > 90 are false and a
	// NaN would sail straight through them. strconv.ParseFloat, which
	// encoding/xml calls, accepts the literal "NaN" without complaint.
	if math.IsNaN(raw.Lat) || math.IsNaN(raw.Lon) {
		return Point{}, fmt.Errorf("%w: coordinates are not numbers", errSkipPoint)
	}
	if raw.Lat == 0 && raw.Lon == 0 {
		// A real position in the Gulf of Guinea, but far more likely a <trkpt>
		// whose lat/lon attributes were absent: encoding/xml leaves a missing
		// attribute at zero and reports nothing.
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
// dropped; a document holding no usable point yields a Track with no segment
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
