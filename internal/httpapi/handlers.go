package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"lolremote/internal/league"
)

func (s *Server) handleAPIGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/state":
		writeJSON(w, 200, s.svc.State())
	case "/api/champions":
		champs, err := s.svc.Champions()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		writeJSON(w, 200, champs)
	case "/api/spells":
		spells, err := s.svc.SummonerSpells()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		writeJSON(w, 200, spells)
	case "/api/runes":
		data, err := s.svc.Runes()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		writeJSON(w, 200, data)
	case "/api/runes/recommended":
		data, err := s.svc.Recommended()
		if err != nil {
			writeJSON(w, 503, errJSON(err))
			return
		}
		writeJSON(w, 200, data)
	default:
		writeJSON(w, 404, map[string]interface{}{"error": "not found"})
	}
}

func readBody(r *http.Request) map[string]interface{} {
	var body map[string]interface{}
	defer r.Body.Close()
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body == nil {
		body = map[string]interface{}{}
	}
	return body
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func asBool(v interface{}) bool {
	b, _ := v.(bool)
	return b
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func (s *Server) handleAPIPost(w http.ResponseWriter, r *http.Request) {
	body := readBody(r)
	switch r.URL.Path {
	case "/api/select":
		err := s.svc.Select(asInt(body["actionId"]), asInt(body["championId"]), asBool(body["lock"]))
		if err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/spells":
		if err := s.svc.SetSpells(asInt(body["spell1"]), asInt(body["spell2"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/swaps/request":
		if err := s.svc.RequestSwap(asString(body["type"]), asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/swaps/accept":
		if err := s.svc.AcceptSwap(asString(body["type"]), asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/swaps/decline":
		if err := s.svc.DeclineSwap(asString(body["type"]), asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/swaps/cancel":
		if err := s.svc.CancelSwap(asString(body["type"]), asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/accept":
		if err := s.svc.Accept(); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/decline":
		if err := s.svc.Decline(); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/auto-accept":
		enabled := asBool(body["enabled"])
		s.svc.SetAutoAccept(enabled)
		writeJSON(w, 200, map[string]interface{}{"ok": true, "autoAccept": enabled})

	case "/api/runes/activate":
		if err := s.svc.ActivatePage(asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/runes/delete":
		if err := s.svc.DeletePage(asInt(body["id"])); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/runes/save":
		fields, err := league.PageFieldsFrom(body)
		if err != nil {
			writeJSON(w, 400, errJSON(err))
			return
		}
		var pageID *int
		if id := asInt(body["id"]); id != 0 {
			pageID = &id
		}
		if err := s.svc.SavePage(fields, pageID); err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})

	case "/api/runes/apply":
		fields, err := league.PageFieldsFrom(body)
		if err != nil {
			writeJSON(w, 400, errJSON(err))
			return
		}
		name, err := s.svc.ApplyPage(fields)
		if err != nil {
			s.writeAPIErr(w, err)
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true, "page": name})

	default:
		writeJSON(w, 404, map[string]interface{}{"error": "not found"})
	}
}

// writeAPIErr maps a league error to the HTTP status the mobile page
// expects: a RemoteError or ValidationError is the user's fault (400) with
// a readable message, anything else is a 503 (LCU unreachable, etc).
func (s *Server) writeAPIErr(w http.ResponseWriter, err error) {
	var remoteErr *league.RemoteError
	if errors.As(err, &remoteErr) {
		msg := extractMessage(remoteErr.Body)
		if msg == "" {
			msg = fmt.Sprintf("Error %d", remoteErr.Code)
		}
		writeJSON(w, 400, map[string]interface{}{"error": msg})
		return
	}
	var validationErr *league.ValidationError
	if errors.As(err, &validationErr) {
		writeJSON(w, 400, errJSON(err))
		return
	}
	writeJSON(w, 503, errJSON(err))
}

func extractMessage(body []byte) string {
	var v map[string]interface{}
	if json.Unmarshal(body, &v) == nil {
		if m, ok := v["message"].(string); ok {
			return m
		}
	}
	return string(body)
}
