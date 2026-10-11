package league

import (
	"strconv"
	"strings"
)

// myAction returns the local player's pending pick or ban action, if any.
func myAction(session map[string]interface{}) map[string]interface{} {
	me := asInt(session["localPlayerCellId"])
	var mine []map[string]interface{}
	for _, a := range actions(session) {
		if asInt(a["actorCellId"]) == me && !asBool(a["completed"]) {
			mine = append(mine, a)
		}
	}
	for _, a := range mine {
		if asBool(a["isInProgress"]) {
			return a
		}
	}
	for _, a := range mine {
		if asString(a["type"]) == "pick" {
			return a
		}
	}
	return nil
}

func actions(s map[string]interface{}) []map[string]interface{} {
	var out []map[string]interface{}
	for _, group := range asList(s["actions"]) {
		for _, item := range asList(group) {
			out = append(out, asMap(item))
		}
	}
	return out
}

// buildTeams assembles picks, hovers and bans for both teams.
func buildTeams(s map[string]interface{}) Teams {
	me := asInt(s["localPlayerCellId"])
	acts := actions(s)

	locked := map[int]bool{}
	picking := map[int]bool{}
	for _, a := range acts {
		if asString(a["type"]) == "pick" && asBool(a["completed"]) {
			locked[asInt(a["actorCellId"])] = true
		}
		if asBool(a["isInProgress"]) {
			picking[asInt(a["actorCellId"])] = true
		}
	}

	slot := func(c map[string]interface{}, ally bool) TeamSlot {
		intent := 0
		if ally {
			intent = asInt(c["championPickIntent"])
		}
		cellID := asInt(c["cellId"])
		return TeamSlot{
			Champion: asInt(c["championId"]),
			Intent:   intent,
			Position: strings.ToLower(asString(c["assignedPosition"])),
			Name:     asString(c["gameName"]),
			Me:       cellID == me,
			Locked:   locked[cellID],
			Picking:  picking[cellID],
		}
	}

	doneBans := func(ally bool) []int {
		out := []int{}
		for _, a := range acts {
			if asString(a["type"]) == "ban" && asBool(a["completed"]) {
				if cid := asInt(a["championId"]); cid != 0 && asBool(a["isAllyAction"]) == ally {
					out = append(out, cid)
				}
			}
		}
		return out
	}

	bans := asMap(s["bans"])
	allyBans := intSliceOrFallback(bans["myTeamBans"], doneBans(true))
	enemyBans := intSliceOrFallback(bans["theirTeamBans"], doneBans(false))

	allies := []TeamSlot{}
	for _, c := range asList(s["myTeam"]) {
		allies = append(allies, slot(asMap(c), true))
	}
	enemies := []TeamSlot{}
	for _, c := range asList(s["theirTeam"]) {
		enemies = append(enemies, slot(asMap(c), false))
	}

	return Teams{
		Allies: allies, Enemies: enemies,
		AllyBans: filterPositive(allyBans), EnemyBans: filterPositive(enemyBans),
	}
}

// State reports the current phase and, during champ select, the pending
// action, allowed champions, timer and both teams.
func (s *Service) State() StateResult {
	phaseRaw, err := s.client.Get("/lol-gameflow/v1/gameflow-phase")
	if err != nil {
		return StateResult{Connected: false, AutoAccept: s.AutoAccept()}
	}
	phase := asString(phaseRaw)
	result := StateResult{Connected: true, Phase: phase, AutoAccept: s.AutoAccept()}
	if phase != "ChampSelect" {
		return result
	}

	raw, err := s.client.Get("/lol-champ-select/v1/session")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	sess := asMap(raw)
	me := asInt(sess["localPlayerCellId"])
	myCell := map[string]interface{}{}
	for _, c := range asList(sess["myTeam"]) {
		cm := asMap(c)
		if asInt(cm["cellId"]) == me {
			myCell = cm
			break
		}
	}
	timer := asMap(sess["timer"])

	sel := &SelectState{
		TimeLeft:       asInt(timer["adjustedTimeLeftInPhase"]) / 1000,
		LockedChampion: asInt(myCell["championId"]),
		Allowed:        []int{},
		Teams:          buildTeams(sess),
	}

	if a := myAction(sess); a != nil {
		kind := asString(a["type"])
		sel.Action = &Action{
			ID: asInt(a["id"]), Type: kind,
			InProgress: asBool(a["isInProgress"]),
			ChampionID: asInt(a["championId"]),
		}
		if kind == "ban" {
			sel.Allowed = s.banAllowed(sess)
		} else if allowedRaw, err := s.client.Get("/lol-champ-select/v1/pickable-champion-ids"); err == nil {
			sel.Allowed = toIntSlice(allowedRaw)
		}
	}

	result.Select = sel
	return result
}

// banAllowed lists every champion not already banned or locked in, since
// the LCU's own bannable-champion-ids endpoint isn't reliable.
func (s *Service) banAllowed(sess map[string]interface{}) []int {
	taken := map[int]bool{}
	for _, a := range actions(sess) {
		if asBool(a["completed"]) {
			if cid := asInt(a["championId"]); cid != 0 {
				taken[cid] = true
			}
		}
	}
	bans := asMap(sess["bans"])
	for _, cid := range toIntSlice(bans["myTeamBans"]) {
		taken[cid] = true
	}
	for _, cid := range toIntSlice(bans["theirTeamBans"]) {
		taken[cid] = true
	}
	champs, _ := s.Champions()
	allowed := []int{}
	for _, c := range champs {
		if !taken[c.ID] {
			allowed = append(allowed, c.ID)
		}
	}
	return allowed
}

// Select hovers (and, if lock is set, locks in or bans) a champion for the
// given pending action id.
func (s *Service) Select(actionID, championID int, lock bool) error {
	actionPath := actionPathFor(actionID)
	if _, err := s.client.Patch(actionPath, map[string]interface{}{"championId": championID}); err != nil {
		return err
	}
	if lock {
		if _, err := s.client.Patch(actionPath, map[string]interface{}{"championId": championID, "completed": true}); err != nil {
			return err
		}
	}
	return nil
}

func actionPathFor(id int) string {
	return "/lol-champ-select/v1/session/actions/" + strconv.Itoa(id)
}
