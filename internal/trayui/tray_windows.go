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
func Run(pin, url string) {
	systray.Run(func() { onReady(pin, url) }, func() {})
}

func onReady(pin, url string) {
	systray.SetIcon(icon.ICO(32))
	systray.SetTitle("LoL Remote")
	systray.SetTooltip("LoL Remote — PIN " + pin)

	show := systray.AddMenuItem("Afficher", "Affiche l'adresse et le PIN")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quitter", "Arrête LoL Remote")

	// Affichée immédiatement au lancement : on ne veut pas que l'utilisateur
	// ait à deviner qu'il faut ouvrir le menu de l'icône pour voir le PIN.
	showPopup(pin, url)

	for {
		select {
		case <-show.ClickedCh:
			showPopup(pin, url)
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
func showPopup(pin, url string) {
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

		w := webview.NewWithOptions(webview.WebViewOptions{
			WindowOptions: webview.WindowOptions{
				Title: "LoL Remote", Width: 360, Height: 560, Center: true,
			},
		})
		if w == nil {
			messageBox("LoL Remote", fmt.Sprintf("Adresse : %s\nCode PIN : %s", url, pin))
			return
		}
		defer w.Destroy()

		w.SetHtml(renderPopup(pin, url))
		w.Run()
	}()
}

func renderPopup(pin, url string) string {
	qrB64 := ""
	if png, err := qrcode.Encode(url, qrcode.Medium, 220); err == nil {
		qrB64 = base64.StdEncoding.EncodeToString(png)
	}
	html := popupTemplate
	html = strings.ReplaceAll(html, "__URL__", url)
	html = strings.ReplaceAll(html, "__PIN__", pin)
	html = strings.ReplaceAll(html, "__QR__", qrB64)
	return html
}

// ReportFatal shows an error dialog — there is no console to print to.
func ReportFatal(msg string) {
	messageBox("LoL Remote — Erreur", msg)
}

func messageBox(title, text string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	_, _, _ = proc.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), 0)
}
