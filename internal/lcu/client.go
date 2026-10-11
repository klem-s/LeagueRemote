// Package lcu is the infrastructure adapter that talks to the real League
// client over its local HTTPS API. It implements the league.Client port;
// nothing in internal/league imports this package.
package lcu

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"lolremote/internal/league"
)

type creds struct {
	port string
	pass string
}

var (
	portRe  = regexp.MustCompile(`--app-port=(\d+)`)
	tokenRe = regexp.MustCompile(`--remoting-auth-token=([\w-]+)`)
)

// Client is a league.Client backed by the local League client's LCU API.
type Client struct {
	mu    sync.Mutex
	creds *creds
	http  *http.Client
}

func New() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// readCredsFromLockfile parses a League lockfile (format
// LeagueClient:pid:port:password:protocol). Only used when the user
// explicitly points LOL_REMOTE_LOCKFILE at one.
func readCredsFromLockfile(path string) (*creds, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	if len(parts) < 4 {
		return nil, errors.New("lockfile invalide")
	}
	return &creds{port: parts[2], pass: parts[3]}, nil
}

// readCredsFromProcess finds the running client's port and auth token via
// its own command line, so it works regardless of where League is
// installed (unlike a hardcoded lockfile path).
func readCredsFromProcess() (*creds, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-CimInstance Win32_Process -Filter "name='LeagueClientUx.exe'" | Select-Object -ExpandProperty CommandLine`,
	).Output()
	if err != nil {
		return nil, err
	}
	s := string(out)
	pm := portRe.FindStringSubmatch(s)
	tm := tokenRe.FindStringSubmatch(s)
	if pm == nil || tm == nil {
		return nil, errors.New("client League introuvable")
	}
	return &creds{port: pm[1], pass: tm[1]}, nil
}

func readCreds() (*creds, error) {
	if path := os.Getenv("LOL_REMOTE_LOCKFILE"); path != "" {
		if c, err := readCredsFromLockfile(path); err == nil {
			return c, nil
		}
	}
	return readCredsFromProcess()
}

func (c *Client) request(method, path string, body interface{}, raw bool) (interface{}, []byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		c.mu.Lock()
		if c.creds == nil || attempt == 1 {
			if cr, err := readCreds(); err == nil {
				c.creds = cr
			} else {
				c.creds = nil
			}
		}
		cr := c.creds
		c.mu.Unlock()

		if cr == nil {
			return nil, nil, errors.New("Client League introuvable")
		}

		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
		req, err := http.NewRequest(method, "https://127.0.0.1:"+cr.port+path, reqBody)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("riot:"+cr.pass)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = errors.New("client League injoignable")
			c.mu.Lock()
			c.creds = nil
			c.mu.Unlock()
			if attempt == 0 {
				continue
			}
			return nil, nil, lastErr
		}
		content, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			c.mu.Lock()
			c.creds = nil
			c.mu.Unlock()
			if attempt == 0 {
				continue
			}
			return nil, nil, &league.RemoteError{Code: resp.StatusCode, Body: content}
		}
		if resp.StatusCode >= 400 {
			return nil, nil, &league.RemoteError{Code: resp.StatusCode, Body: content}
		}
		if raw {
			return nil, content, nil
		}
		if len(content) == 0 {
			return nil, nil, nil
		}
		var v interface{}
		if err := json.Unmarshal(content, &v); err != nil {
			return nil, nil, err
		}
		return v, nil, nil
	}
	return nil, nil, lastErr
}

func (c *Client) Get(path string) (interface{}, error) {
	v, _, err := c.request("GET", path, nil, false)
	return v, err
}
func (c *Client) Post(path string, body interface{}) (interface{}, error) {
	v, _, err := c.request("POST", path, body, false)
	return v, err
}
func (c *Client) Put(path string, body interface{}) (interface{}, error) {
	v, _, err := c.request("PUT", path, body, false)
	return v, err
}
func (c *Client) Patch(path string, body interface{}) (interface{}, error) {
	v, _, err := c.request("PATCH", path, body, false)
	return v, err
}
func (c *Client) Delete(path string) (interface{}, error) {
	v, _, err := c.request("DELETE", path, nil, false)
	return v, err
}
func (c *Client) Raw(path string) ([]byte, error) {
	_, raw, err := c.request("GET", path, nil, true)
	return raw, err
}

var _ league.Client = (*Client)(nil)
