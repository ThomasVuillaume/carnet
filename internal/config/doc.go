/*
Package config loads, completes and validates carnet.yaml.

It is the trust boundary of the pipeline: once Load returns, no other package
re-checks a required key or a numeric bound. Rejections split in two: a
ValidationErrors names the faulty keys and maps to CLI exit code 2, while a
document the decoder itself refuses — malformed, or carrying an unknown key —
maps to exit code 1.

It knows nothing of the filesystem beyond the reader it is handed. Existence of
the GPX file, of the photo directory and of the caption file is the check
command's business, not this package's.
*/
package config
