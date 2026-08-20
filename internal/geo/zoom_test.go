package geo

import "testing"

// TestSelectZoom compares exactly. A zoom level is a whole number taken off a
// fixed ladder, so there is nothing here to grant a tolerance to.
//
// Each row below quotes the zoom level each axis allows, as a fraction. "Width
// allows 11.231" means the box would exactly fill the available width at that
// imaginary level, and anything above it overflows. SelectZoom returns the
// smaller of the two figures, rounded down. They come from
// log2(available pixels / (span * TileSize)), computed outside this package, so
// the table does not check the implementation against itself.
//
// A row only tells a correct implementation from a broken one when the two axes
// round down to different levels. Rows where both round to the same level still
// earn their place — they check the overall scale — and say so.
//
// The first four rows are one recorded ride seen through four viewports. The
// rest are shapes a recorded track never has: no width, no height at all, or
// the whole planet.
//
// Checked by mutation. Six ways of breaking the code are each caught by at
// least one row: reversing the height subtraction, dropping the TileSize
// factor, running the loop upwards, making either axis comparison always true,
// taking latitude in raw degrees, and counting padding once instead of twice.
// The last two are caught by a single row each — 60N and the padding row — so
// neither can be dropped as redundant.
func TestSelectZoom(t *testing.T) {
	t.Parallel()

	// Bounding box of testdata/gpx/Flashbird_2026-07-27_10-06.gpx: 2399 points
	// in one segment, a Cévennes ride around 44°N. Written out as four numbers
	// rather than parsed from the file, because gpx imports geo and importing
	// gpx here would close the cycle. Checking that these numbers survive the
	// trip from file to SelectZoom belongs to a test where the pipeline is
	// assembled.
	cevennes := Bounds{MinLat: 44.189456, MinLon: 3.146918, MaxLat: 44.492176, MaxLon: 3.858531}

	tests := []struct {
		name                            string
		b                               Bounds
		width, height, padding, maxZoom int
		want                            int
	}{
		{
			// Width allows 11.231 and height 11.090. Both round down to 11, so
			// this row cannot say which axis SelectZoom listened to. What it
			// does check is the scale: a missing TileSize factor, or a loop run
			// upwards, lands nowhere near 11.
			name:  "recorded ride on the default map",
			b:     cevennes,
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 11,
		},
		{
			// A shorter image. Height now allows only 10.500 where width still
			// allows 11.231, and the two round down to different levels. This
			// is the row that fails when the height is computed as
			// northY - southY: that subtraction comes out negative, so every
			// comparison against it passes, the vertical constraint is never
			// really applied, and the answer quietly becomes 11.
			name:  "recorded ride on a short map",
			b:     cevennes,
			width: 1280, height: 500,
			padding: 32, maxZoom: 15,
			want: 10,
		},
		{
			// The mirror case: width allows 9.751, height 11.980. It fails if
			// the width comparison is dropped or always true.
			name:  "recorded ride on a tall map",
			b:     cevennes,
			width: 500, height: 1280,
			padding: 32, maxZoom: 15,
			want: 9,
		},
		{
			// The same image as the first row, whose answer was 11. Only
			// maxZoom changes, so this row fails if the ceiling is ignored.
			name:  "recorded ride under a maxZoom ceiling",
			b:     cevennes,
			width: 1280, height: 720,
			padding: 32, maxZoom: 10,
			want: 10,
		},
		{
			// A box with neither width nor height fits at every level, so the
			// ceiling is the answer. A track trimmed down to a single point
			// arrives here. It is also the input on which the closed form,
			// log2 of a ratio, would divide zero by zero.
			name:  "single point",
			b:     Bounds{MinLat: 45, MinLon: 5, MaxLat: 45, MaxLon: 5},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 15,
		},
		{
			// Both corners share a longitude, so the box has no width and
			// height decides alone. An out-and-back along one road gets close
			// to this.
			name:  "north-south track only",
			b:     Bounds{MinLat: 44, MinLon: 5, MaxLat: 45, MaxLon: 5},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 9,
		},
		{
			// The other way round: both corners share a latitude, so width
			// decides alone.
			name:  "east-west track only",
			b:     Bounds{MinLat: 45, MinLon: 4, MaxLat: 45, MaxLon: 6},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 9,
		},
		// The same 1° by 1° box twice, once at the equator and once at 60°N. It
		// is square in degrees, but Mercator stretches a degree of latitude by
		// 1/cos φ, so the northern one comes out 2.03 times taller in pixels
		// and settles one level lower. Take the height from raw degrees instead
		// of from Project and both answer 9, so the 60N row is the one that
		// fails. None of the recorded-ride rows above catch that mistake: raw
		// degrees are 39.8% off there, which is not enough to change the level.
		{
			name:  "one degree square at the equator",
			b:     Bounds{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 9,
		},
		{
			name:  "one degree square at 60N",
			b:     Bounds{MinLat: 60, MinLon: 0, MaxLat: 61, MaxLon: 1},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 8,
		},
		{
			// The whole Mercator square still fits at level 1, its 512 px
			// against the 656 the padding leaves. Reaching level 0 would take
			// an axis narrower than TileSize, which the 320 px minimum image
			// size that config enforces very nearly rules out.
			name:  "whole world",
			b:     Bounds{MinLat: -MaxLatitude, MinLon: -180, MaxLat: MaxLatitude, MaxLon: 180},
			width: 1280, height: 720,
			padding: 32, maxZoom: 15,
			want: 1,
		},
		{
			// 500 px of padding on each of the two horizontal edges leaves
			// -280 px of height. Nothing fits, not even level 0, and SelectZoom
			// returns the bottom of the ladder rather than a negative level.
			// config rejects a padding like this before SelectZoom ever sees
			// it, so the row checks the fallback, not the validation. It is
			// also the only row that fails if padding is counted once per axis
			// instead of once per edge.
			name:  "padding leaves no room",
			b:     cevennes,
			width: 1280, height: 720,
			padding: 500, maxZoom: 15,
			want: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := SelectZoom(tc.b, tc.width, tc.height, tc.padding, tc.maxZoom)
			if got != tc.want {
				t.Errorf("SelectZoom(%+v, %d, %d, %d, %d) = %d, want %d",
					tc.b, tc.width, tc.height, tc.padding, tc.maxZoom, got, tc.want)
			}
		})
	}
}
