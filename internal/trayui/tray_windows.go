// Package trayui is the Windows-only presentation layer for LoL Remote:
// a system tray icon instead of a console window, since the app ships
// built with -H=windowsgui. It only knows plain strings (the PIN and the
// URL to display) — no dependency on internal/league or internal/httpapi.
package trayui

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/getlantern/systray"
	webview "github.com/jchv/go-webview2"
	qrcode "github.com/skip2/go-qrcode"

	"lolremote/internal/icon"
)

//go:embed popup.html
var popupTemplate string

var popupOpenMu sync.Mutex
var popupOpen bool

// Run shows the tray icon and blocks forever running its message loop.
func Run(pin, url string, update Update) {
	systray.Run(func() { onReady(pin, url, update) }, func() {})
}

func onReady(pin, url string, update Update) {
	systray.SetIcon(icon.ICO(32))
	systray.SetTitle("LoL Remote")
	tooltip := "LoL Remote — PIN " + pin
	if update.Tag != "" {
		tooltip += " (update available)"
	}
	systray.SetTooltip(tooltip)

	show := systray.AddMenuItem("Show", "Show the address and PIN")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Stop LoL Remote")

	// Shown immediately at launch: the user shouldn't have to guess that
	// the tray menu is where the PIN lives.
	showPopup(pin, url, update)

	for {
		select {
		case <-show.ClickedCh:
			showPopup(pin, url, update)
		case <-quit.ClickedCh:
			systray.Quit()
			os.Exit(0)
		}
	}
}

// showPopup opens the styled info window in its own locked OS thread (Win32
// windows are thread-affine: creation and message loop must share a
// thread). Falls back to a plain MessageBox if WebView2 isn't available —
// e.g. an unmaintained Windows install without the Evergreen runtime.
func showPopup(pin, url string, update Update) {
	popupOpenMu.Lock()
	if popupOpen {
		popupOpenMu.Unlock()
		return
	}
	popupOpen = true
	popupOpenMu.Unlock()

	go func() {
		defer func() {
			popupOpenMu.Lock()
			popupOpen = false
			popupOpenMu.Unlock()
		}()
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		const popupWidth, popupHeight = 360, 560

		w := webview.NewWithOptions(webview.WebViewOptions{
			WindowOptions: webview.WindowOptions{
				Title: "LoL Remote", Width: popupWidth, Height: popupHeight, Center: true,
			},
		})
		if w == nil {
			text := fmt.Sprintf("Address: %s\nPIN code: %s", url, pin)
			if update.Tag != "" {
				text += fmt.Sprintf("\n\nUpdate available: %s\n%s", update.Tag, update.URL)
			}
			messageBox("LoL Remote", text)
			return
		}
		defer w.Destroy()

		// HintFixed also re-measures the window so popupWidth/popupHeight
		// become the exact *client* area (the initial size above is the
		// outer window, shrunk by the title bar) — without this the content
		// was a few pixels taller than the viewport and triggered a
		// scrollbar. It also drops the resize/maximize affordances.
		w.SetSize(popupWidth, popupHeight, webview.HintFixed)

		// Links in the popup open in the system browser instead of
		// navigating the popup itself away from its own content.
		_ = w.Bind("openurl", func(target string) { _ = exec.Command("cmd", "/c", "start", "", target).Start() })

		w.SetHtml(renderPopup(pin, url, update))
		w.Run()
	}()
}

func renderPopup(pin, url string, update Update) string {
	qrB64 := ""
	if png, err := qrcode.Encode(url, qrcode.Medium, 220); err == nil {
		qrB64 = base64.StdEncoding.EncodeToString(png)
	}
	updateBlock := ""
	if update.Tag != "" {
		updateBlock = fmt.Sprintf(
			`<button class="update" onclick="window.openurl('%s')">Update available: %s</button>`,
			strings.ReplaceAll(update.URL, "'", ""), update.Tag,
		)
	}
	html := popupTemplate
	html = strings.ReplaceAll(html, "__URL__", url)
	html = strings.ReplaceAll(html, "__PIN__", pin)
	html = strings.ReplaceAll(html, "__QR__", qrB64)
	html = strings.ReplaceAll(html, "__UPDATE__", updateBlock)
	return html
}

// ReportFatal shows an error dialog — there is no console to print to.
func ReportFatal(msg string) {
	messageBox("LoL Remote — Error", msg)
}

func messageBox(title, text string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	_, _, _ = proc.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), 0)
}
