package league

import "testing"

func TestSummonerSpellsFiltersAndSorts(t *testing.T) {
	classic := []interface{}{"CLASSIC"}
	client := &fakeClient{
		responses: map[string]interface{}{
			"/lol-game-data/assets/v1/summoner-spells.json": []interface{}{
				map[string]interface{}{"id": float64(4), "name": "Flash", "gameModes": classic},
				map[string]interface{}{"id": float64(11), "name": "Smite", "gameModes": classic},
				map[string]interface{}{"id": float64(0), "name": "Unselectable", "gameModes": classic},            // id<=0: dropped
				map[string]interface{}{"id": float64(99), "name": "", "gameModes": classic},                       // no name: dropped
				map[string]interface{}{"id": float64(32), "name": "Mark", "gameModes": []interface{}{"ARAM"}},     // not CLASSIC: dropped
				map[string]interface{}{"id": float64(2202), "name": "Flee", "gameModes": []interface{}{"CHERRY"}}, // Arena: dropped
				map[string]interface{}{"id": float64(14), "name": "Ignite", "gameModes": classic},
			},
		},
	}
	svc := NewService(client)

	spells, err := svc.SummonerSpells()
	if err != nil {
		t.Fatalf("SummonerSpells() error: %v", err)
	}
	want := []SummonerSpell{{ID: 4, Name: "Flash"}, {ID: 14, Name: "Ignite"}, {ID: 11, Name: "Smite"}}
	if len(spells) != len(want) {
		t.Fatalf("got %+v, want %+v", spells, want)
	}
	for i := range want {
		if spells[i] != want[i] {
			t.Errorf("spells[%d] = %+v, want %+v", i, spells[i], want[i])
		}
	}
}

func TestSetSpellsPatchesMySelection(t *testing.T) {
	client := &fakeClient{}
	svc := NewService(client)

	if err := svc.SetSpells(4, 11); err != nil {
		t.Fatalf("SetSpells() error: %v", err)
	}
	if len(client.patches) != 1 {
		t.Fatalf("got %d Patch calls, want 1", len(client.patches))
	}
	got := client.patches[0]
	if got.path != "/lol-champ-select/v1/session/my-selection" {
		t.Errorf("path = %q", got.path)
	}
	body, ok := got.body.(map[string]interface{})
	if !ok {
		t.Fatalf("body = %+v, want map", got.body)
	}
	if body["spell1Id"] != 4 || body["spell2Id"] != 11 {
		t.Errorf("body = %+v, want spell1Id=4 spell2Id=11", body)
	}
}
