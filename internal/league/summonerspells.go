package league

import (
	"sort"
	"strings"
)

// SummonerSpells returns the spells valid on Summoner's Rift (ranked,
// normals, etc.) — filtered using each entry's own gameModes list rather
// than a hardcoded ID allowlist, so ARAM-only (Mark), Arena-only and other
// special-mode spells are excluded. Cached for the life of the service.
func (s *Service) SummonerSpells() ([]SummonerSpell, error) {
	s.spellsMu.Lock()
	defer s.spellsMu.Unlock()
	if s.spellsCache != nil {
		return s.spellsCache, nil
	}

	data, err := s.client.Get("/lol-game-data/assets/v1/summoner-spells.json")
	if err != nil {
		return nil, err
	}
	out := []SummonerSpell{}
	for _, item := range asList(data) {
		c := asMap(item)
		id := asInt(c["id"])
		name := asString(c["name"])
		if id <= 0 || name == "" || !hasGameMode(c, "CLASSIC") {
			continue
		}
		// The actual iconPath from the data, same as Perk/Style icons in
		// runes.go — not a guessed URL pattern like champion-icons/{id}.png.
		out = append(out, SummonerSpell{ID: id, Name: name, Icon: asString(c["iconPath"])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	s.spellsCache = out
	return out, nil
}

// hasGameMode checks a game-data entry's own "gameModes" list — "CLASSIC"
// is Summoner's Rift (covers ranked, normals, quickplay), which is what
// excludes ARAM-only (e.g. Mark), Arena-only and other special-mode spells.
func hasGameMode(c map[string]interface{}, mode string) bool {
	for _, m := range asList(c["gameModes"]) {
		if strings.EqualFold(asString(m), mode) {
			return true
		}
	}
	return false
}

// SetSpells assigns both summoner spell slots at once — the LCU's
// my-selection endpoint only accepts them together, not one at a time.
func (s *Service) SetSpells(spell1, spell2 int) error {
	_, err := s.client.Patch("/lol-champ-select/v1/session/my-selection",
		map[string]interface{}{"spell1Id": spell1, "spell2Id": spell2})
	return err
}
