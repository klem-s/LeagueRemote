package league

import "sort"

// SummonerSpells returns every summoner spell from the game-data file,
// cached for the life of the service. Unlike Champions(), this isn't
// filtered down to what's valid for the current queue/game mode — that
// isn't exposed anywhere obvious, so an invalid pick here is expected to
// surface as a real error from SetSpells instead of being guessed at.
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
		if id <= 0 || name == "" {
			continue
		}
		out = append(out, SummonerSpell{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	s.spellsCache = out
	return out, nil
}

// SetSpells assigns both summoner spell slots at once — the LCU's
// my-selection endpoint only accepts them together, not one at a time.
func (s *Service) SetSpells(spell1, spell2 int) error {
	_, err := s.client.Patch("/lol-champ-select/v1/session/my-selection",
		map[string]interface{}{"spell1Id": spell1, "spell2Id": spell2})
	return err
}
