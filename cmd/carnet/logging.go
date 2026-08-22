package main

import (
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
)

// newLogger builds the root logger. Every stage of the pipeline receives a
// child of it through With, never a logger of its own.
//
// w is the stderr threaded through run, not os.Stderr: a test hands it a
// buffer and reads the lines back.
//
// TextHandler rather than JSONHandler — carnet is read by a human at a
// terminal and nothing ingests these lines. The choice is cheap to revisit:
// call sites name attributes, never formats, so a CI build that wants JSON
// changes this line and nothing else.
func newLogger(w io.Writer, verbose bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level(verbose),
		// Under --verbose only. AddSource calls runtime.Callers for every
		// record it keeps, and the emitting line is worth that price just
		// when someone is reading debug output.
		AddSource:   verbose,
		ReplaceAttr: replaceAttr,
	}

	return slog.New(slog.NewTextHandler(w, opts))
}

// level maps --verbose to the threshold the handler keeps.
//
// Info by default: the pipeline milestones and the report of check. Debug adds
// the per-stage detail — chosen zoom, cache hits, correlation deltas.
//
// Info rather than Warn as the floor means the lines proving the privacy purge
// ran stay visible without --verbose, which is what makes them evidence.
func level(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// replaceAttr rewrites the two attributes the handler adds by itself. It is
// the single ReplaceAttr the handler calls, dispatching by key so each rewrite
// stays one idea.
//
// Both rewrites serve the same end: a line that depends only on the run, never
// on the clock or the machine. Two runs of the same trip then write identical
// logs, which turns a diff of two stderr captures into a debugging tool.
//
// Attributes inside a group are returned untouched. ReplaceAttr visits those
// too, and a photo group carrying its own "time" or "source" key is the
// caller's data, not the handler's.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}

	switch a.Key {
	// A run lasts seconds, so the absolute hour teaches nothing while costing
	// ~35 columns ahead of the message. Stages that are actually slow — tile
	// download, rendering — log an explicit duration attribute instead. The
	// zero Attr is the documented way to drop one.
	case slog.TimeKey:
		return slog.Attr{}
	case slog.SourceKey:
		return shortenSource(a)
	default:
		return a
	}
}

// shortenSource cuts the absolute path AddSource records down to its last two
// segments: /home/someone/carnet/internal/gpx/parse.go:97 becomes
// gpx/parse.go:97.
//
// The recorded path is the one the build machine saw, home directory included.
// Two segments rather than one because parse.go alone repeats across packages.
//
// The value carried under SourceKey is a *slog.Source. Anything else means a
// caller chose that key for its own attribute, so it passes through — a typed
// nil included, which satisfies the assertion and would panic on src.File.
func shortenSource(a slog.Attr) slog.Attr {
	src, ok := a.Value.Any().(*slog.Source)
	if !ok || src == nil {
		return a
	}

	dir, file := filepath.Split(src.File)
	short := filepath.Join(filepath.Base(dir), file)

	return slog.String(slog.SourceKey, short+":"+strconv.Itoa(src.Line))
}
