//go:build !windows

// This fallback exists only so the project still builds and vets on a
// non-Windows dev machine (no GTK/cgo toolchain needed). LoL Remote itself
// is Windows-only — this is never what ships.
package trayui

import "fmt"

func Run(pin, url string, update Update) {
	fmt.Println()
	fmt.Println("  LoL Remote started")
	fmt.Println("  On your phone:", url)
	fmt.Println("  PIN code     :", pin)
	fmt.Println()
	fmt.Println("  (Ctrl+C to stop)")
	if update.Tag != "" {
		fmt.Println("  Update available:", update.Tag, update.URL)
	}
	select {}
}

func ReportFatal(msg string) {
	fmt.Println("Error:", msg)
}
