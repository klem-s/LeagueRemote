package league

import "testing"

type fakeClient struct {
	responses map[string]interface{}
}

func (f *fakeClient) Get(path string) (interface{}, error)           { return f.responses[path], nil }
func (f *fakeClient) Post(string, interface{}) (interface{}, error)  { return nil, nil }
func (f *fakeClient) Put(string, interface{}) (interface{}, error)   { return nil, nil }
func (f *fakeClient) Patch(string, interface{}) (interface{}, error) { return nil, nil }
func (f *fakeClient) Delete(string) (interface{}, error)             { return nil, nil }
func (f *fakeClient) Raw(string) ([]byte, error)                     { return nil, nil }

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

// TestStatePlanning checks the Planning flag the mobile page uses to decide
// whether a pre-turn Hover is allowed: true when timer.phase is PLANNING,
// false otherwise — even if the player's own action isn't in progress
// either way, since "no action in progress" alone doesn't mean planning.
func TestStatePlanning(t *testing.T) {
	baseSession := func(phase string) map[string]interface{} {
		return map[string]interface{}{
			"localPlayerCellId": float64(1),
			"myTeam": []interface{}{
				map[string]interface{}{"cellId": float64(1), "championId": float64(0), "gameName": "Me"},
			},
			"theirTeam": []interface{}{},
			"actions": []interface{}{
				[]interface{}{
					map[string]interface{}{"id": float64(20), "actorCellId": float64(1), "type": "pick", "isInProgress": false, "completed": false},
				},
			},
			"bans":  map[string]interface{}{},
			"timer": map[string]interface{}{"adjustedTimeLeftInPhase": float64(30000), "phase": phase},
		}
	}

	for _, tc := range []struct {
		phase string
		want  bool
	}{
		{"PLANNING", true},
		{"BAN_PICK", false},
		{"FINALIZATION", false},
		{"", false},
	} {
		client := &fakeClient{responses: map[string]interface{}{
			"/lol-gameflow/v1/gameflow-phase": "ChampSelect",
			"/lol-champ-select/v1/session":    baseSession(tc.phase),
		}}
		svc := NewService(client)
		result := svc.State()
		if result.Select == nil {
			t.Fatalf("phase %q: expected a select state", tc.phase)
		}
		if result.Select.Planning != tc.want {
			t.Errorf("phase %q: Planning = %v, want %v", tc.phase, result.Select.Planning, tc.want)
		}
	}
}

// TestBuildTeamsTrades checks that a trade entry from the session is
// attached to the matching ally's cell, and never to "me" or to enemies —
// trades are only ever offered between allies.
func TestBuildTeamsTrades(t *testing.T) {
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
	}

	teams := buildTeams(session)

	if teams.Allies[0].TradeID != 7 || teams.Allies[0].TradeState != "RECEIVED" {
		t.Errorf("Ally0: got TradeID=%d TradeState=%q, want 7/RECEIVED", teams.Allies[0].TradeID, teams.Allies[0].TradeState)
	}
	if teams.Allies[1].TradeState != "" {
		t.Errorf("Me: got TradeState=%q, want empty (no self-trade)", teams.Allies[1].TradeState)
	}
	if teams.Enemies[0].TradeState != "" {
		t.Errorf("Enemy0: got TradeState=%q, want empty (enemies never trade)", teams.Enemies[0].TradeState)
	}
}
