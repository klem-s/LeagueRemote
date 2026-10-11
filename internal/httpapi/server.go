// Package httpapi is the interface-adapter layer: it translates HTTP
// requests into calls on a league.Service and serializes the results back
// to JSON for the phone. It knows nothing about the LCU or PowerShell —
// only internal/league's exported API.
package httpapi

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"path"
	"strconv"
	"strings"

	"lolremote/internal/icon"
	"lolremote/internal/league"
)

//go:embed page.html
var pageHTML string

// Server holds everything the HTTP layer needs: the use-case service, the
// PIN phones must present on /api/*, and the embedded mobile page.
type Server struct {
	svc     *league.Service
	pin     string
	version string
}

func NewServer(svc *league.Service, pin, version string) *Server {
	return &Server{svc: svc, pin: pin, version: version}
}

func (s *Server) checkPin(r *http.Request) bool {
	return r.Header.Get("X-Pin") == s.pin
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeBytes(w http.ResponseWriter, code int, ctype string, b []byte, cache bool) {
	w.Header().Set("Content-Type", ctype)
	if cache {
		w.Header().Set("Cache-Control", "max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(code)
	w.Write(b)
}

func errJSON(err error) map[string]interface{} {
	return map[string]interface{}{"error": err.Error()}
}

var ctypeBySuffix = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".svg": "image/svg+xml",
}

func ctypeFor(p string) string {
	if t, ok := ctypeBySuffix[strings.ToLower(path.Ext(p))]; ok {
		return t
	}
	return "image/png"
}

func (s *Server) serveAsset(w http.ResponseWriter, p string) {
	b, err := s.svc.Asset(p)
	if err != nil {
		writeBytes(w, 404, "text/plain", []byte{}, false)
		return
	}
	writeBytes(w, 200, ctypeFor(p), b, true)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	page := strings.ReplaceAll(pageHTML, "__VERSION__", s.version)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		writeBytes(w, 200, "text/html; charset=utf-8", []byte(page), false)
	})

	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(icon.Manifest())
		writeBytes(w, 200, "application/manifest+json", b, false)
	})
	mux.HandleFunc("/icon-192.png", func(w http.ResponseWriter, r *http.Request) {
		writeBytes(w, 200, "image/png", icon.PNG(192), false)
	})
	mux.HandleFunc("/icon-512.png", func(w http.ResponseWriter, r *http.Request) {
		writeBytes(w, 200, "image/png", icon.PNG(512), false)
	})

	mux.HandleFunc("/icon/", func(w http.ResponseWriter, r *http.Request) {
		cid := strings.TrimPrefix(r.URL.Path, "/icon/")
		if _, err := strconv.Atoi(cid); err != nil {
			writeBytes(w, 404, "text/plain", []byte{}, false)
			return
		}
		s.serveAsset(w, "/lol-game-data/assets/v1/champion-icons/"+cid+".png")
	})

	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("p")
		if !strings.HasPrefix(p, "/lol-game-data/assets/") || strings.Contains(p, "..") {
			writeBytes(w, 404, "text/plain", []byte{}, false)
			return
		}
		s.serveAsset(w, p)
	})

	// Temporary debugging aid, not pin-gated so it's reachable by just
	// typing the address in any browser: dumps the raw champ-select
	// session JSON. See league.Service.RawSession.
	mux.HandleFunc("/debug/session", func(w http.ResponseWriter, r *http.Request) {
		raw, err := s.svc.RawSession()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		b, _ := json.MarshalIndent(raw, "", "  ")
		writeBytes(w, 200, "application/json", b, false)
	})

	// Same idea, for the Live Client Data API (in-game stats) instead of
	// champ-select. See league.Service.RawLiveGame.
	mux.HandleFunc("/debug/livegame", func(w http.ResponseWriter, r *http.Request) {
		raw, err := s.svc.RawLiveGame()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		b, _ := json.MarshalIndent(raw, "", "  ")
		writeBytes(w, 200, "application/json", b, false)
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if !s.checkPin(r) {
			writeJSON(w, 401, map[string]interface{}{"error": "Incorrect PIN"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAPIGet(w, r)
		case http.MethodPost:
			s.handleAPIPost(w, r)
		default:
			writeJSON(w, 404, map[string]interface{}{"error": "not found"})
		}
	})

	return mux
}
