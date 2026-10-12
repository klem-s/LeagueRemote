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
//
// The delay uses "ping", not "timeout": timeout.exe checks whether stdin
// is an actual console and aborts immediately with "Input redirection is
// not supported" if not — which is exactly the case here, since this app
// is a -H=windowsgui build with no console of its own. That made the
// delay a no-op (cmd's "&" runs the next command regardless of the
// previous one's exit code), silently recreating the original race. ping
// has no such console dependency — the classic, reliable way to sleep in
// a batch script for precisely this reason.
func RunInstaller(path string) error {
	cmd := exec.Command("cmd", "/c",
		fmt.Sprintf(`ping -n 3 127.0.0.1 >nul & "%s" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART`, path))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
