package geo

// Bounds is a geographic lat/lon rectangle, in decimal degrees.
type Bounds struct {
	MinLat, MinLon float64
	MaxLat, MaxLon float64
}
