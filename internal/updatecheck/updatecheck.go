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

// RunInstaller launches a downloaded installer fully silently — no wizard,
// no progress window, nothing to click through. /VERYSILENT suppresses all
// UI, /SUPPRESSMSGBOXES auto-answers anything that would otherwise be a
// message box, /NORESTART skips any reboot prompt (irrelevant here but
// harmless). The installer's own [Run] entry relaunches the app once done.
// The caller is expected to exit this process shortly after — the
// installer can't overwrite lol-remote.exe while it's still running.
func RunInstaller(path string) error {
	return exec.Command(path, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART").Start()
}
