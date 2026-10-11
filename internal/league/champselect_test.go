package league

import "testing"

type patchCall struct {
	path string
	body interface{}
}

type fakeClient struct {
	responses    map[string]interface{}
	liveGameData interface{}
	patches      []patchCall
}

func (f *fakeClient) Get(path string) (interface{}, error)          { return f.responses[path], nil }
func (f *fakeClient) Post(string, interface{}) (interface{}, error) { return nil, nil }
func (f *fakeClient) Put(string, interface{}) (interface{}, error)  { return nil, nil }
func (f *fakeClient) Patch(path string, body interface{}) (interface{}, error) {
	f.patches = append(f.patches, patchCall{path, body})
	return nil, nil
}
func (f *fakeClient) Delete(string) (interface{}, error) { return nil, nil }
func (f *fakeClient) Raw(string) ([]byte, error)         { return nil, nil }
func (f *fakeClient) LiveGameData() (interface{}, error) { return f.liveGameData, nil }

// TestStateInProgressReportsLiveStats checks that once the game itself is
// running, State() reports the local player's live line from the Live
// Client Data API instead of trying to parse a champ-select session that no
// longer exists.
func TestStateInProgressReportsLiveStats(t *testing.T) {
	client := &fakeClient{
		responses: map[string]interface{}{"/lol-gameflow/v1/gameflow-phase": "InProgress"},
		liveGameData: map[string]interface{}{
			"gameData": map[string]interface{}{"gameTime": float64(600)},
			"activePlayer": map[string]interface{}{
				"riotIdGameName": "Me", "riotIdTagLine": "EUW",
			},
			"allPlayers": []interface{}{
				map[string]interface{}{
					"riotIdGameName": "Me", "riotIdTagLine": "EUW",
					"championName": "Ahri", "level": float64(11),
					"scores": map[string]interface{}{
						"kills": float64(4), "deaths": float64(1), "assists": float64(6), "creepScore": float64(150),
					},
				},
				map[string]interface{}{
					"riotIdGameName": "Someone", "riotIdTagLine": "EUW",
					"championName": "Zed", "level": float64(10),
					"scores": map[string]interface{}{"kills": float64(2), "deaths": float64(3), "assists": float64(1), "creepScore": float64(90)},
				},
			},
		},
	}
	svc := NewService(client)
	result := svc.State()

	if result.Phase != "InProgress" {
		t.Errorf("Phase = %q, want InProgress", result.Phase)
	}
	if result.Select != nil {
		t.Error("Select should be nil while in game")
	}
	if result.Live == nil {
		t.Fatal("Live is nil, want the local player's live stats")
	}
	want := LiveStats{GameTime: 600, Champion: "Ahri", Level: 11, Kills: 4, Deaths: 1, Assists: 6, CS: 150, CSPerMin: 15}
	if *result.Live != want {
		t.Errorf("Live = %+v, want %+v", *result.Live, want)
	}
}

// TestStatePicking checks that the player whose action is currently
// isInProgress is reported as Picking, and nobody else is — this is what
// the mobile page highlights to answer "who is picking right now".
func TestStatePicking(t *testing.T) {
	session := map[string]interface{}{
		"localPlayerCellId": float64(1),
		"myTeam": []interface{}{
			map[string]interface{}{"cellId": float64(0), "championId": float64(0), "gameName": "Ally0", "assignedPosition": "top"},
			map[string]interface{}{"cellId": float64(1), "championId": float64(0), "gameName": "Me", "assignedPosition": "middle"},
		},
		"theirTeam": []interface{}{
			map[string]interface{}{"cellId": float64(5), "championId": float64(0), "gameName": "Enemy0", "assignedPosition": "jungle"},
		},
		"actions": []interface{}{
			[]interface{}{
				map[string]interface{}{"id": float64(10), "actorCellId": float64(0), "type": "pick", "isInProgress": true, "completed": false},
				map[string]interface{}{"id": float64(11), "actorCellId": float64(1), "type": "pick", "isInProgress": false, "completed": false},
				map[string]interface{}{"id": float64(12), "actorCellId": float64(5), "type": "pick", "isInProgress": false, "completed": false},
			},
		},
		"bans":  map[string]interface{}{},
		"timer": map[string]interface{}{"adjustedTimeLeftInPhase": float64(15000)},
	}

	client := &fakeClient{responses: map[string]interface{}{
		"/lol-gameflow/v1/gameflow-phase":            "ChampSelect",
		"/lol-champ-select/v1/session":               session,
		"/lol-champ-select/v1/pickable-champion-ids": []interface{}{},
	}}
	svc := NewService(client)

	result := svc.State()
	if result.Select == nil {
		t.Fatal("expected a select state during ChampSelect")
	}
	teams := result.Select.Teams

	if !teams.Allies[0].Picking {
		t.Errorf("Ally0 (cell 0, isInProgress) should be Picking, got false")
	}
	if teams.Allies[1].Picking {
		t.Errorf("Me (cell 1, not isInProgress) should not be Picking")
	}
	if teams.Enemies[0].Picking {
		t.Errorf("Enemy0 (cell 5, not isInProgress) should not be Picking")
	}
}

// TestStateUpcomingPickDuringBan pins down the actual pre-pick bug: while
// the local player's own ban is in progress, Action must be that ban (so
// the ban UI/lock-in still works), but UpcomingPick must be their separate
// pending pick action — modeled directly on a real captured session where
// this distinction was missing and pre-picking silently hovered the ban
// instead of the pick.
func TestStateUpcomingPickDuringBan(t *testing.T) {
	session := map[string]interface{}{
		"localPlayerCellId": float64(0),
		"myTeam": []interface{}{
			map[string]interface{}{"cellId": float64(0), "championId": float64(0), "gameName": "Me"},
		},
		"theirTeam": []interface{}{},
		"actions": []interface{}{
			[]interface{}{
				map[string]interface{}{"id": float64(0), "actorCellId": float64(0), "type": "ban", "isInProgress": true, "completed": false, "championId": float64(12)},
			},
			[]interface{}{
				map[string]interface{}{"id": float64(1), "actorCellId": float64(0), "type": "pick", "isInProgress": false, "completed": false},
			},
		},
		"bans":  map[string]interface{}{},
		"timer": map[string]interface{}{"adjustedTimeLeftInPhase": float64(20000), "phase": "BAN_PICK"},
	}

	client := &fakeClient{responses: map[string]interface{}{
		"/lol-gameflow/v1/gameflow-phase": "ChampSelect",
		"/lol-champ-select/v1/session":    session,
	}}
	result := NewService(client).State()
	if result.Select == nil {
		t.Fatal("expected a select state")
	}

	if result.Select.Action == nil || result.Select.Action.Type != "ban" || result.Select.Action.ID != 0 {
		t.Errorf("Action = %+v, want the in-progress ban (id 0)", result.Select.Action)
	}
	if result.Select.UpcomingPick == nil || result.Select.UpcomingPick.ID != 1 {
		t.Fatalf("UpcomingPick = %+v, want the pending pick (id 1)", result.Select.UpcomingPick)
	}
	if result.Select.UpcomingPick.InProgress {
		t.Error("UpcomingPick.InProgress = true, want false (it's not this player's active turn)")
	}
}

// TestBuildTeamsSwaps checks that champion/position/pick-order swap entries
// are attached to the matching ally's cell as Swaps, filtered to only
// actionable states, and never attached to "me" or to enemies.
func TestBuildTeamsSwaps(t *testing.T) {
	session := map[string]interface{}{
		"localPlayerCellId": float64(1),
		"myTeam": []interface{}{
			map[string]interface{}{"cellId": float64(0), "gameName": "Ally0"},
			map[string]interface{}{"cellId": float64(1), "gameName": "Me"},
		},
		"theirTeam": []interface{}{
			map[string]interface{}{"cellId": float64(5), "gameName": "Enemy0"},
		},
		"actions": []interface{}{},
		"bans":    map[string]interface{}{},
		"trades": []interface{}{
			map[string]interface{}{"id": float64(7), "cellId": float64(0), "state": "RECEIVED"},
			// An entry keyed by my own cellId or an enemy's should never
			// surface — the LCU shouldn't send these, but don't trust it.
			map[string]interface{}{"id": float64(8), "cellId": float64(1), "state": "AVAILABLE"},
			map[string]interface{}{"id": float64(9), "cellId": float64(5), "state": "AVAILABLE"},
		},
		"positionSwaps": []interface{}{
			map[string]interface{}{"id": float64(4), "cellId": float64(0), "state": "AVAILABLE"},
		},
		"pickOrderSwaps": []interface{}{
			// INVALID is not actionable — must be filtered out entirely.
			map[string]interface{}{"id": float64(2), "cellId": float64(0), "state": "INVALID"},
		},
	}

	teams := buildTeams(session)

	want := []SwapOption{
		{Type: "champion", ID: 7, State: "RECEIVED"},
		{Type: "position", ID: 4, State: "AVAILABLE"},
	}
	got := teams.Allies[0].Swaps
	if len(got) != len(want) {
		t.Fatalf("Ally0: got %d swaps %+v, want %+v", len(got), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Ally0 swap[%d]: got %+v, want %+v", i, got[i], want[i])
		}
	}

	if len(teams.Allies[1].Swaps) != 0 {
		t.Errorf("Me: got Swaps=%+v, want none (no self-swap)", teams.Allies[1].Swaps)
	}
	if len(teams.Enemies[0].Swaps) != 0 {
		t.Errorf("Enemy0: got Swaps=%+v, want none (enemies never swap)", teams.Enemies[0].Swaps)
	}
}

// TestBuildTeamsSwapsRealSession locks in parsing against an actual LCU
// session dump (trimmed to the relevant fields), captured live via
// /debug/session during a real Normal Draft champ select — not a guess.
func TestBuildTeamsSwapsRealSession(t *testing.T) {
	session := map[string]interface{}{
		"localPlayerCellId": float64(9),
		"myTeam": []interface{}{
			map[string]interface{}{"cellId": float64(5), "gameName": "", "assignedPosition": "bottom"},
			map[string]interface{}{"cellId": float64(6), "gameName": "", "assignedPosition": "jungle"},
			map[string]interface{}{"cellId": float64(7), "gameName": "", "assignedPosition": "utility"},
			map[string]interface{}{"cellId": float64(8), "gameName": "", "assignedPosition": "top"},
			map[string]interface{}{"cellId": float64(9), "gameName": "Klem", "assignedPosition": "middle"},
		},
		"theirTeam": []interface{}{
			map[string]interface{}{"cellId": float64(0)}, map[string]interface{}{"cellId": float64(1)},
			map[string]interface{}{"cellId": float64(2)}, map[string]interface{}{"cellId": float64(3)},
			map[string]interface{}{"cellId": float64(4)},
		},
		"actions": []interface{}{},
		"bans":    map[string]interface{}{},
		"trades": []interface{}{
			map[string]interface{}{"cellId": float64(7), "id": float64(4), "state": "INVALID"},
			map[string]interface{}{"cellId": float64(8), "id": float64(5), "state": "INVALID"},
			map[string]interface{}{"cellId": float64(6), "id": float64(6), "state": "INVALID"},
			map[string]interface{}{"cellId": float64(5), "id": float64(7), "state": "INVALID"},
		},
		"positionSwaps": []interface{}{
			map[string]interface{}{"cellId": float64(7), "id": float64(4), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(8), "id": float64(5), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(6), "id": float64(6), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(5), "id": float64(7), "state": "AVAILABLE"},
		},
		"pickOrderSwaps": []interface{}{
			map[string]interface{}{"cellId": float64(7), "id": float64(4), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(8), "id": float64(5), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(6), "id": float64(6), "state": "AVAILABLE"},
			map[string]interface{}{"cellId": float64(5), "id": float64(7), "state": "AVAILABLE"},
		},
	}

	teams := buildTeams(session)

	// Cell 7 (utility ally): champion trade is INVALID (nobody's locked in
	// yet) so it must be dropped, but position and pick-order swaps are
	// both AVAILABLE and must both show up.
	var utility TeamSlot
	for _, a := range teams.Allies {
		if a.Position == "utility" {
			utility = a
		}
	}
	want := []SwapOption{
		{Type: "position", ID: 4, State: "AVAILABLE"},
		{Type: "pickOrder", ID: 4, State: "AVAILABLE"},
	}
	if len(utility.Swaps) != len(want) {
		t.Fatalf("utility ally: got %d swaps %+v, want %+v", len(utility.Swaps), utility.Swaps, want)
	}
	for i := range want {
		if utility.Swaps[i] != want[i] {
			t.Errorf("utility ally swap[%d]: got %+v, want %+v", i, utility.Swaps[i], want[i])
		}
	}

	// "Me" (cellId 9, no entries in any of the three arrays at all) must
	// have no swaps.
	for _, a := range teams.Allies {
		if a.Me && len(a.Swaps) != 0 {
			t.Errorf("me: got Swaps=%+v, want none", a.Swaps)
		}
	}
}
