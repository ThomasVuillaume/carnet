# CLAUDE.md

`carnet` is a CLI static generator for motorcycle trip logs: it turns a trip
directory (GPX track + photos) into publish-ready static content (map PNG,
elevation SVG, Markdown/JSON) for an Astro site, with privacy scrubbing.

**`/.private/PRD-carnet.md` is the authoritative spec.** When in doubt about behavior,
defaults, formats, or scope, read it — do not invent. This file only condenses
the non-negotiable rules.

## Language policy

- Code, identifiers, comments, config keys, commit messages: **English**.
- README and user-facing docs: **French**.

## Hard constraints

### Dependencies (enforced by the project's Definition of Done)

`go.mod` may only ever contain: the standard library, `golang.org/x/image`,
one EXIF library (open decision D1 in the PRD, not chosen yet — isolate it
behind an interface), and `gopkg.in/yaml.v3`.

**Never add**, even to save time: `flopp/go-staticmaps`, `fogleman/gg`,
`tkrajina/gpxgo`, or any geometry library. GPX parsing uses `encoding/xml`;
Douglas-Peucker, Haversine, and Web Mercator are written by hand — learning
these is half the point of the project.

### Determinism

Same input ⇒ byte-for-byte identical output. No generation timestamps, no
random identifiers, file traversal explicitly sorted, output order never
derived from map iteration or goroutine scheduling.

### Code style

- Go ≥ 1.26. `golangci-lint` must pass with zero warnings.
- Errors wrapped with `%w`; exported error types for cases callers must
  distinguish (invalid config vs. privacy violation).
- No `panic` outside `main`. No mutable global state.
- `context.Context` propagated through anything that blocks or does network I/O.
- Logging via `log/slog` to stderr; level controlled by `--verbose`.

## Privacy is a security requirement

Pipeline order is fixed: exclusion zones → trim start/end → coordinate
rounding → EXIF purge. GPS EXIF tags are **never** kept in output files, even
if listed in `keep_exif_tags` — hard-coded rule. A final verification pass
re-reads written files and exits with code 3 if any GPS tag or excluded
coordinate leaks. Photos inside an exclusion zone are dropped entirely, not
just stripped of position.

## Out of scope (v1)

No server, no GUI, no AVIF/WebP encoding (Astro's pipeline does that), no
routing engine, no vector tiles, no database, no network calls other than the
tile server. Re-read PRD section 3 before adding any feature.

## Commands

```sh
go build ./...
go test -race ./...        # full suite must run offline and finish < 60 s
golangci-lint run
```

Coverage must stay above 85% on `internal/geo`, `internal/gpx`,
`internal/privacy`.

## Testing rules

- No test may touch the network; test tiles are served from `embed.FS`
  (`testdata/tiles/`) behind the `tiles.Source` interface.
- Rendering uses golden files (`testdata/golden/`), regenerated with `-update`,
  compared pixel-by-pixel with tolerance.
- Determinism test: run generation twice, compare SHA-256 of all outputs.
- Privacy regression test: assert no GPS tag and no excluded coordinate in
  outputs.

## Architecture

- `cmd/carnet/` — entry point, flag parsing, wiring; the only place `panic`
  or `os.Exit` is allowed.
- `internal/config/` — `carnet.yaml` loading and validation.
- `internal/gpx/` — GPX parsing (`encoding/xml`), track model, simplification.
- `internal/geo/` — Mercator, Haversine, bounding box, zoom selection.
- `internal/tiles/` — tile source interface, disk cache, bounded concurrent
  download (context-cancellable, rate-limited to 2 req/s), assembly.
- `internal/render/` — antialiased polyline (`x/image/vector`), attribution,
  elevation SVG.
- `internal/photos/` — EXIF read, photo/track correlation, renaming.
- `internal/privacy/` — exclusion zones, trimming, rounding, EXIF purge,
  final leak check.
- `internal/output/` — Markdown frontmatter, JSON, asset copy, alt texts.

CLI exit codes: `0` success, `1` runtime error, `2` invalid config,
`3` privacy violation.
