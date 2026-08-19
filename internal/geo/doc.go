/*
Package geo holds the hand-written geometry carnet runs on: Haversine
distance, Web Mercator projection, bounding boxes, zoom selection and
Douglas-Peucker simplification.

It speaks only in numbers — coordinates, pixels, indices — and imports no
other internal package. Anything that knows what a track or a photo is
belongs one layer up. That rule is what lets internal/gpx and internal/render
both depend on geo without a cycle.
*/
package geo
