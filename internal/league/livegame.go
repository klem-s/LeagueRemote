package league

// parseLiveStats turns a raw /liveclientdata/allgamedata response into the
// local player's own live line. Confidence note: this endpoint shape is
// officially documented by Riot (unlike most of what this app touches),
// but which field identifies "me" among allPlayers isn't part of that
// documentation, so it's matched defensively — see playerIdentity.
func parseLiveStats(raw interface{}) *LiveStats {
	m := asMap(raw)
	gameTime := asFloat(asMap(m["gameData"])["gameTime"])

	myID := playerIdentity(asMap(m["activePlayer"]))
	var me map[string]interface{}
	if myID != "" {
		for _, pRaw := range asList(m["allPlayers"]) {
			p := asMap(pRaw)
			if playerIdentity(p) == myID {
				me = p
				break
			}
		}
	}
	if me == nil {
		return &LiveStats{GameTime: gameTime}
	}

	scores := asMap(me["scores"])
	cs := asInt(scores["creepScore"])
	var csPerMin float64
	if gameTime > 0 {
		csPerMin = float64(cs) / (gameTime / 60)
	}
	return &LiveStats{
		GameTime: gameTime,
		Champion: asString(me["championName"]),
		Level:    asInt(me["level"]),
		Kills:    asInt(scores["kills"]),
		Deaths:   asInt(scores["deaths"]),
		Assists:  asInt(scores["assists"]),
		CS:       cs,
		CSPerMin: csPerMin,
	}
}

// playerIdentity builds a comparable identity from a Live Client Data
// player object: the modern Riot ID fields where present, falling back to
// the legacy summonerName — the same gameName#tagLine pattern this app
// (and others in this project) has already had to apply to champ-select
// and lobby data.
func playerIdentity(p map[string]interface{}) string {
	if g := asString(p["riotIdGameName"]); g != "" {
		if t := asString(p["riotIdTagLine"]); t != "" {
			return g + "#" + t
		}
		return g
	}
	return asString(p["summonerName"])
}
