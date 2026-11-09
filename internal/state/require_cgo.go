//go:build !cgo

package state

// go-sqlite3 otherwise builds a no-op SQLite stub when CGO is disabled.
// Explicitly fail the build rather than shipping a binary that cannot open its ledger.
// The undefined name is intentional and should appear in the compiler error.
var _ = ERROR_BOXWEAVE_SQLITE_REQUIRES_CGO_ENABLED
