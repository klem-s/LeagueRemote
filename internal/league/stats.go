package league

import (
	"fmt"
	"sort"
)

// Stats are computed client-side from recent match history, not pulled
// from any pre-aggregated endpoint — the LCU doesn't expose one. Confidence
// note: /lol-match-history/v1/products/lol/current-summoner/matches and
// /lol-summoner/v1/current-summoner are both long-standing, widely used by
// other community tools; the match-history response shape followed here
// (participants + participantIdentities, matching MatchDto) is the one
// every such tool relies on.

const (
	defaultStatsCount = 20
	maxStatsCount     = 200 // sane upper bound regardless of what the caller asks for
	statsPageSize     = 20  // fetched one known-safe page at a time rather than one big range
)

func (s *Service) Stats(count int) (StatsResult, error) {
	if count <= 0 {
		count = defaultStatsCount
	}
	if count > maxStatsCount {
		count = maxStatsCount
	}

	meRaw, err := s.client.Get("/lol-summoner/v1/current-summoner")
	if err != nil {
		return StatsResult{}, err
	}
	me := asMap(meRaw)
	myPUUID := asString(me["puuid"])
	myAccountID := asInt(me["accountId"])

	games, err := s.fetchRecentGames(count)
	if err != nil {
		return StatsResult{}, err
	}

	champs, _ := s.Champions()
	nameByID := make(map[int]string, len(champs))
	for _, c := range champs {
		nameByID[c.ID] = c.Name
	}

	perChamp := map[int]*ChampionStats{}
	result := StatsResult{}

	for _, g := range games {
		game := asMap(g)
		pid := findParticipantID(asList(game["participantIdentities"]), myPUUID, myAccountID)
		if pid == 0 {
			continue
		}
		p := findParticipant(asList(game["participants"]), pid)
		if p == nil {
			continue
		}
		stats := asMap(p["stats"])
		champID := asInt(p["championId"])

		cs, ok := perChamp[champID]
		if !ok {
			cs = &ChampionStats{ChampionID: champID, ChampionName: nameByID[champID]}
			perChamp[champID] = cs
		}

		result.Games++
		cs.Games++
		if asBool(stats["win"]) {
			result.Wins++
			cs.Wins++
		}
		result.Kills += asInt(stats["kills"])
		result.Deaths += asInt(stats["deaths"])
		result.Assists += asInt(stats["assists"])
	}

	if result.Games > 0 {
		result.WinRate = float64(result.Wins) / float64(result.Games) * 100
	}
	for _, cs := range perChamp {
		if cs.Games > 0 {
			cs.WinRate = float64(cs.Wins) / float64(cs.Games) * 100
		}
		result.PerChampion = append(result.PerChampion, *cs)
	}
	sort.Slice(result.PerChampion, func(i, j int) bool {
		return result.PerChampion[i].Games > result.PerChampion[j].Games
	})

	return result, nil
}

// fetchRecentGames pages through match history in fixed-size, known-safe
// chunks rather than one big begIndex/endIndex range (no confirmation the
// LCU handles an arbitrarily large single range well), stopping once count
// games have been collected or a short page signals there's no more
// history.
func (s *Service) fetchRecentGames(count int) ([]interface{}, error) {
	var games []interface{}
	for begIndex := 0; len(games) < count; begIndex += statsPageSize {
		endIndex := begIndex + statsPageSize - 1
		raw, err := s.client.Get(fmt.Sprintf(
			"/lol-match-history/v1/products/lol/current-summoner/matches?begIndex=%d&endIndex=%d", begIndex, endIndex))
		if err != nil {
			if begIndex == 0 {
				return nil, err
			}
			break // later pages failing after we already have some games isn't fatal
		}
		page := asList(asMap(asMap(raw)["games"])["games"])
		games = append(games, page...)
		if len(page) < statsPageSize {
			break // short page: reached the end of available history
		}
	}
	if len(games) > count {
		games = games[:count]
	}
	return games, nil
}

// findParticipantID maps "me" to a participantId for this specific game: by
// puuid where the client provides one, falling back to the legacy
// accountId on older match-history payloads.
func findParticipantID(identities []interface{}, puuid string, accountID int) int {
	for _, idRaw := range identities {
		id := asMap(idRaw)
		player := asMap(id["player"])
		if puuid != "" && asString(player["puuid"]) == puuid {
			return asInt(id["participantId"])
		}
		if puuid == "" && accountID != 0 && asInt(player["accountId"]) == accountID {
			return asInt(id["participantId"])
		}
	}
	return 0
}

func findParticipant(participants []interface{}, participantID int) map[string]interface{} {
	for _, pRaw := range participants {
		p := asMap(pRaw)
		if asInt(p["participantId"]) == participantID {
			return p
		}
	}
	return nil
}
