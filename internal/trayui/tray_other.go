//go:build !windows

// This fallback exists only so the project still builds and vets on a
// non-Windows dev machine (no GTK/cgo toolchain needed). LoL Remote itself
// is Windows-only — this is never what ships.
package trayui

import "fmt"

func Run(info Info) {
	fmt.Println()
	fmt.Println("  LoL Remote started", info.Version)
	fmt.Println("  On your phone:", info.URL)
	fmt.Println("  PIN code     :", info.PIN)
	fmt.Println()
	fmt.Println("  (Ctrl+C to stop)")
	if info.Update.Tag != "" {
		fmt.Println("  Update available:", info.Update.Tag, info.Update.URL)
	}
	select {}
}

func ReportFatal(msg string) {
	fmt.Println("Error:", msg)
}
