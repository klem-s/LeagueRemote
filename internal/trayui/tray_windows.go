// Package trayui is the Windows-only presentation layer for LoL Remote:
// a system tray icon instead of a console window, since the app ships
// built with -H=windowsgui. It only knows plain strings (the PIN and the
// URL to display) — no dependency on internal/league or internal/httpapi.
package trayui

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/getlantern/systray"
	qrcode "github.com/skip2/go-qrcode"

	"lolremote/internal/icon"
)

// Run shows the tray icon and blocks forever running its message loop.
func Run(pin, url string) {
	systray.Run(func() { onReady(pin, url) }, func() {})
}

func onReady(pin, url string) {
	systray.SetIcon(icon.ICO(32))
	systray.SetTitle("LoL Remote")
	systray.SetTooltip("LoL Remote — PIN " + pin)

	info := systray.AddMenuItem("Adresse et code PIN", "Affiche l'adresse et le PIN")
	qr := systray.AddMenuItem("Code QR", "Ouvre un code QR vers l'adresse")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quitter", "Arrête LoL Remote")

	// Affichée immédiatement au lancement : on ne veut pas que l'utilisateur
	// ait à deviner qu'il faut ouvrir le menu de l'icône pour voir le PIN.
	go messageBox("LoL Remote", fmt.Sprintf("Adresse : %s\nCode PIN : %s", url, pin))

	go func() {
		for {
			select {
			case <-info.ClickedCh:
				messageBox("LoL Remote", fmt.Sprintf("Adresse : %s\nCode PIN : %s", url, pin))
			case <-qr.ClickedCh:
				showQR(url)
			case <-quit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

// showQR renders the URL as a QR code and opens it in the default image
// viewer — simpler and more robust than drawing a custom window.
func showQR(url string) {
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		messageBox("LoL Remote — Erreur", "Impossible de générer le code QR : "+err.Error())
		return
	}
	f, err := os.CreateTemp("", "lol-remote-qr-*.png")
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(png)
	_ = exec.Command("cmd", "/c", "start", "", f.Name()).Start()
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
