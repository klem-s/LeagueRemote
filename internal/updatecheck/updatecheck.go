// Package updatecheck looks up the latest published GitHub release for a
// repo, so the app can offer a newer build — and, since LoLRemote-Setup.exe
// started being published alongside the portable exe, fetch and launch
// that installer directly instead of just linking to the release page.
package updatecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const installerAssetName = "LoLRemote-Setup.exe"

// HelperFlag, passed as os.Args[1], tells main() to skip the normal app
// startup entirely and instead run RunInstallerHelper — see RunInstaller.
const HelperFlag = "--run-installer"

type Release struct {
	Tag string
	URL string // release page, as a fallback if InstallerURL is empty
	// InstallerURL is the direct download URL for LoLRemote-Setup.exe on
	// this release, or "" if that asset isn't there (e.g. an older release
	// published before the installer existed).
	InstallerURL string
}

// Latest fetches the latest release of "owner/repo" from GitHub's public
// API. No auth needed for a public repo. A short timeout keeps a slow or
// offline network from delaying startup noticeably.
func Latest(ownerRepo string) (Release, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", ownerRepo))
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Release{}, err
	}
	rel := Release{Tag: body.TagName, URL: body.HTMLURL}
	for _, a := range body.Assets {
		if a.Name == installerAssetName {
			rel.InstallerURL = a.BrowserDownloadURL
			break
		}
	}
	return rel, nil
}

// DownloadInstaller fetches the installer from url into the OS temp
// directory, overwriting any previous download there, and returns its path.
func DownloadInstaller(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	dest := filepath.Join(os.TempDir(), installerAssetName)
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", err
	}
	return dest, nil
}

// RunInstaller re-execs the current binary as a detached helper (passing
// HelperFlag and the installer's path) rather than launching the installer
// directly from here. Launching it directly and exiting shortly after
// raced its own AppMutex check against this process's exit — and routing
// the delay through a cmd.exe one-liner to fix that turned out to have its
// own nasty bug: cmd's /c quote-stripping rule breaks once there's more
// than one quoted segment in the command line, which is exactly what
// happened once Go's own argument-escaping wrapped the whole thing in an
// extra outer layer of quotes around the already-quoted installer path.
// With no console to show an error, it just silently did nothing.
//
// Re-execing this same Go binary with a plain argument list sidesteps
// shell quoting entirely — os/exec talks to CreateProcess directly, no
// cmd.exe involved.
func RunInstaller(installerPath string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	return exec.Command(self, HelperFlag, installerPath).Start()
}

// RunInstallerHelper is what main() calls when relaunched with HelperFlag:
// wait for the original process (and the mutex it holds) to be gone, then
// launch the real installer fully silently. This binary is a
// -H=windowsgui build, so neither this relaunch nor the installer it
// starts ever shows a console — nothing to hide, unlike the old cmd.exe
// approach.
func RunInstallerHelper(installerPath string) {
	time.Sleep(2 * time.Second)
	_ = exec.Command(installerPath, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART").Start()
}
