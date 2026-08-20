/*
Package gpx parses GPX 1.1 files into an in-memory track model, using
encoding/xml only.

Simplification lives in internal/geo, not here. Douglas-Peucker measures a
perpendicular distance, which needs an isotropic space, and the PRD expresses
its epsilon in screen pixels at the chosen zoom: both make it a consumer of the
Mercator projection rather than of the GPX model. This package therefore
imports internal/geo, never the reverse.

The model carries no elevation. PRD decision D5 puts altitude out of v1 scope,
the recording device exporting none.
*/
package gpx
