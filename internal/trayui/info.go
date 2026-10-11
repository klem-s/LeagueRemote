package trayui

// Update describes a newer release to point the user at. A zero value
// means "no update available" — the popup shows nothing extra.
type Update struct {
	Tag string // e.g. "v0.3.0"
	URL string // release page to open
}

// Info is everything the tray UI needs to display.
type Info struct {
	PIN     string
	URL     string
	Version string // "dev" for local builds; see main.version
	Update  Update
}
