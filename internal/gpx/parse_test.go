package gpx

import (
	"encoding/xml"
	"errors"
	"io"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

// wrap builds a minimal GPX document around raw <trkpt> markup, keeping table
// entries readable.
func wrap(body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk><name>t</name><trkseg>` + body + `</trkseg></trk>
</gpx>`
}

func TestPointFrom(t *testing.T) {
	t.Parallel()

	stamp := "2026-07-27T08:07:08.000Z"
	want := time.Date(2026, 7, 27, 8, 7, 8, 0, time.UTC)

	tests := []struct {
		name     string
		raw      xmlTrkpt
		wantSkip bool
		wantTime time.Time
	}{
		{
			name:     "nominal point",
			raw:      xmlTrkpt{Lat: 44.472103, Lon: 3.858475, Time: stamp},
			wantTime: want,
		},
		{
			name: "missing timestamp is kept without one",
			raw:  xmlTrkpt{Lat: 44.472103, Lon: 3.858475},
		},
		{
			name:     "unreadable timestamp is a defect",
			raw:      xmlTrkpt{Lat: 44.472103, Lon: 3.858475, Time: "27/07/2026 08:07"},
			wantSkip: true,
		},
		{
			// Every comparison against NaN is false, so the range checks alone
			// would let this through.
			name:     "NaN latitude",
			raw:      xmlTrkpt{Lat: math.NaN(), Lon: 3.858475, Time: stamp},
			wantSkip: true,
		},
		{
			name:     "NaN longitude",
			raw:      xmlTrkpt{Lat: 44.472103, Lon: math.NaN(), Time: stamp},
			wantSkip: true,
		},
		{
			name:     "infinite latitude",
			raw:      xmlTrkpt{Lat: math.Inf(1), Lon: 3.858475, Time: stamp},
			wantSkip: true,
		},
		{
			// Absent lat/lon attributes reach us as zeroes, indistinguishable
			// from a genuine position off the coast of Ghana.
			name:     "both coordinates zero",
			raw:      xmlTrkpt{Time: stamp},
			wantSkip: true,
		},
		{
			name:     "latitude above the pole",
			raw:      xmlTrkpt{Lat: 90.1, Lon: 3.858475, Time: stamp},
			wantSkip: true,
		},
		{
			name:     "longitude past the antimeridian",
			raw:      xmlTrkpt{Lat: 44.472103, Lon: -180.5, Time: stamp},
			wantSkip: true,
		},
		{
			name:     "poles and antimeridian are inclusive",
			raw:      xmlTrkpt{Lat: -90, Lon: 180, Time: stamp},
			wantTime: want,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := pointFrom(tc.raw)

			if tc.wantSkip {
				if !errors.Is(err, errSkipPoint) {
					t.Fatalf("want a skip, got point %+v with err %v", got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Lat != tc.raw.Lat || got.Lon != tc.raw.Lon {
				t.Errorf("coordinates: got (%v, %v), want (%v, %v)",
					got.Lat, got.Lon, tc.raw.Lat, tc.raw.Lon)
			}
			if !got.Time.Equal(tc.wantTime) {
				t.Errorf("timestamp: got %v, want %v", got.Time, tc.wantTime)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		doc          string
		wantSegments int
		wantPoints   int
	}{
		{
			name: "two points",
			doc: wrap(`
			  <trkpt lat="44.47" lon="3.85"><time>2026-07-27T08:07:08Z</time></trkpt>
			  <trkpt lat="44.48" lon="3.86"><time>2026-07-27T08:07:09Z</time></trkpt>`),
			wantSegments: 1,
			wantPoints:   2,
		},
		{
			// An itinerary export carries no <time> at all: it must still yield
			// a mappable track.
			name: "no timestamps anywhere",
			doc: wrap(`
			  <trkpt lat="44.47" lon="3.85"/>
			  <trkpt lat="44.48" lon="3.86"/>`),
			wantSegments: 1,
			wantPoints:   2,
		},
		{
			name:         "a segment left empty by skipping is dropped",
			doc:          wrap(`<trkpt lat="0" lon="0"><time>2026-07-27T08:07:08Z</time></trkpt>`),
			wantSegments: 0,
		},
		{
			name:         "empty segment",
			doc:          wrap(``),
			wantSegments: 0,
		},
		{
			name:         "single point",
			doc:          wrap(`<trkpt lat="44.47" lon="3.85"/>`),
			wantSegments: 1,
			wantPoints:   1,
		},
		{
			name:         "no track at all",
			doc:          `<gpx version="1.1"></gpx>`,
			wantSegments: 0,
		},
		{
			// The Garmin extension blocks in the real fixture must be ignored,
			// not rejected.
			name: "unknown extensions are ignored",
			doc: wrap(`<trkpt lat="44.47" lon="3.85">
			    <time>2026-07-27T08:07:08Z</time>
			    <extensions><gpxtpx:TrackPointExtension xmlns:gpxtpx="urn:x">
			      <gpxtpx:speed>0.97</gpxtpx:speed>
			    </gpxtpx:TrackPointExtension></extensions>
			  </trkpt>`),
			wantSegments: 1,
			wantPoints:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			track, err := Parse(strings.NewReader(tc.doc))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(track.Segments) != tc.wantSegments {
				t.Fatalf("segments: got %d, want %d", len(track.Segments), tc.wantSegments)
			}

			total := 0
			for _, seg := range track.Segments {
				total += len(seg.Points)
			}
			if total != tc.wantPoints {
				t.Errorf("points: got %d, want %d", total, tc.wantPoints)
			}
		})
	}
}

func TestParseMalformedXML(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader(`<gpx><trk><trkseg>`))

	if _, ok := errors.AsType[*SyntaxError](err); !ok {
		t.Fatalf("want a *SyntaxError, got %T: %v", err, err)
	}
	// Unwrap must keep the decoder's own error reachable, so that a caller can
	// tell a truncated download from invalid markup.
	_, isXMLSyntax := errors.AsType[*xml.SyntaxError](err)
	if !errors.Is(err, io.ErrUnexpectedEOF) && !isXMLSyntax {
		t.Errorf("the cause is no longer reachable through the chain: %v", err)
	}
}

func TestParseRealTrack(t *testing.T) {
	t.Parallel()

	f, err := os.Open("../../testdata/gpx/Flashbird_2026-07-27_10-06.gpx")
	if err != nil {
		t.Fatalf("fixture unreadable: %v", err)
	}
	defer f.Close() //nolint:errcheck // read-only file

	track, err := Parse(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(track.Segments) != 1 {
		t.Fatalf("segments: got %d, want 1", len(track.Segments))
	}
	// The fixture holds 2399 <trkpt>, every one of them usable.
	if got := len(track.Segments[0].Points); got != 2399 {
		t.Errorf("points: got %d, want 2399", got)
	}

	first := track.Segments[0].Points[0]
	want := Point{Lat: 44.472103, Lon: 3.858475, Time: time.Date(2026, 7, 27, 8, 7, 8, 0, time.UTC)}
	if first.Lat != want.Lat || first.Lon != want.Lon || !first.Time.Equal(want.Time) {
		t.Errorf("first point: got %+v, want %+v", first, want)
	}
}
