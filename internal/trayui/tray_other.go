//go:build !windows

// This fallback exists only so the project still builds and vets on a
// non-Windows dev machine (no GTK/cgo toolchain needed). LoL Remote itself
// is Windows-only — this is never what ships.
package trayui

import "fmt"

func Run(pin, url string) {
	fmt.Println()
	fmt.Println("  LoL Remote lancé")
	fmt.Println("  Sur ton téléphone :", url)
	fmt.Println("  Code PIN          :", pin)
	fmt.Println()
	fmt.Println("  (Ctrl+C pour arrêter)")
	select {}
}

func ReportFatal(msg string) {
	fmt.Println("Erreur:", msg)
}
