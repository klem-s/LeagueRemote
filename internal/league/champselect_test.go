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
