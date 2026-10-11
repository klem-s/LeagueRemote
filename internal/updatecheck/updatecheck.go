// Package updatecheck looks up the latest published GitHub release for a
// repo, so the app can point the user at a newer build. Nothing more —
// no downloading or replacing the running executable.
package updatecheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Release struct {
	Tag string
	URL string
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
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Release{}, err
	}
	return Release{Tag: body.TagName, URL: body.HTMLURL}, nil
}
