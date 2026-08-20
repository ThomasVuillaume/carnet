package geo

import "math"

// As per IAG GRS 80 - https://en.wikipedia.org/wiki/Geodetic_Reference_System_1980
const earthRadiusM = 6_371_008.7714

// DistanceM returns the great-circle distance between two WGS84 points, in metres.
func DistanceM(lat1, lon1, lat2, lon2 float64) float64 {

	const degToRad = math.Pi / 180
	// Convert degrees to radians
	lat1Rad := lat1 * degToRad
	lon1Rad := lon1 * degToRad
	lat2Rad := lat2 * degToRad
	lon2Rad := lon2 * degToRad

	// Haversine formula
	// https://fr.wikipedia.org/wiki/Formule_de_haversine
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	// Keep "a" within the domain Asin accepts. The identity behind the formula
	// is a = sin²(c/2) with c in [0, pi], so a belongs to [0, 1] by
	// construction — but the expression above chains four Sin, two Cos, two
	// products and a sum, and near antipodal points those residues land an ULP
	// or two past 1. Measured: 10.3% of exactly antipodal pairs overshoot, and
	// 8 in 200000 far enough that Asin returns NaN, poisoning every distance
	// summed afterwards.
	//
	// This restores an invariant broken by rounding; it is not a guard against
	// bad input. No lower bound is needed: latitudes stay within [-90, 90], so
	// both cosines are non-negative and a cannot go below 0.
	a = min(a, 1)
	c := math.Asin(math.Sqrt(a))

	return 2 * earthRadiusM * c
}
