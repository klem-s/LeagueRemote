//go:build !windows

package singleinstance

// Acquire always succeeds outside Windows — this is a dev-build fallback,
// LoL Remote itself is Windows-only.
func Acquire() bool { return true }
