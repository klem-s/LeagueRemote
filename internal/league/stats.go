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

const defaultStatsCount = 20

func (s *Service) Stats(count int) (StatsResult, error) {
	if count <= 0 {
		count = defaultStatsCount
	}

	meRaw, err := s.client.Get("/lol-summoner/v1/current-summoner")
	if err != nil {
		return StatsResult{}, err
	}
	me := asMap(meRaw)
	myPUUID := asString(me["puuid"])
	myAccountID := asInt(me["accountId"])

	historyRaw, err := s.client.Get(fmt.Sprintf(
		"/lol-match-history/v1/products/lol/current-summoner/matches?begIndex=0&endIndex=%d", count-1))
	if err != nil {
		return StatsResult{}, err
	}
	games := asList(asMap(asMap(historyRaw)["games"])["games"])

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
