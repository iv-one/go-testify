package gotestify

import (
	"bytes"
	"log/slog"
)

// CleanupT is the subset of *testing.T the capture needs.
type CleanupT interface {
	Cleanup(func())
}

// CaptureSlog routes the default slog logger into the returned buffer for the
// rest of the test, restoring the previous logger at cleanup. Not safe under
// t.Parallel: the default logger is process-global.
func CaptureSlog(t CleanupT) *bytes.Buffer {
	buf := &bytes.Buffer{}
	previous := slog.Default()
	// Debug: the default handler level is Info, so a helper left at nil
	// silently drops the very lines a test capturing slog is usually written
	// to assert on.
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return buf
}
