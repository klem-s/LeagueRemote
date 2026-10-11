package league

import "sort"

// Champions returns the deduplicated champion list, cached for the life of
// the service. The game-data file has special-mode variants sharing a name
// with a different id; we keep the smallest id, which is the real champion.
func (s *Service) Champions() ([]Champion, error) {
	s.champsMu.Lock()
	defer s.champsMu.Unlock()
	if s.champsCache != nil {
		return s.champsCache, nil
	}

	data, err := s.client.Get("/lol-game-data/assets/v1/champion-summary.json")
	if err != nil {
		return nil, err
	}
	byName := map[string]Champion{}
	for _, item := range asList(data) {
		c := asMap(item)
		id := asInt(c["id"])
		if id <= 0 {
			continue
		}
		name := asString(c["name"])
		if existing, ok := byName[name]; !ok || id < existing.ID {
			byName[name] = Champion{ID: id, Name: name}
		}
	}
	out := make([]Champion, 0, len(byName))
	for _, c := range byName {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	s.champsCache = out
	return out, nil
}
