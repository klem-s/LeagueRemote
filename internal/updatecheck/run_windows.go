//go:build windows

package updatecheck

import (
	"fmt"
	"os/exec"
	"syscall"
)

// RunInstaller launches a downloaded installer fully silently — no wizard,
// no progress window, nothing to click through. /VERYSILENT suppresses all
// UI, /SUPPRESSMSGBOXES auto-answers anything that would otherwise be a
// message box, /NORESTART skips any reboot prompt (irrelevant here but
// harmless). The installer's own [Run] entry relaunches the app once done.
//
// The install itself is deferred by a couple seconds, run through a hidden
// cmd.exe wrapper detached from this process — launching the installer
// directly and then exiting raced its AppMutex check against this
// process's own exit, and with /SUPPRESSMSGBOXES in play, losing that race
// meant the "please close LoL Remote first" prompt got silently
// auto-dismissed as a cancel: the app closed (this process exited) but
// nothing actually installed. Deferring guarantees this process (and the
// mutex it holds) is long gone before Setup ever checks.
func RunInstaller(path string) error {
	cmd := exec.Command("cmd", "/c",
		fmt.Sprintf(`timeout /t 2 /nobreak >nul & "%s" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART`, path))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
