package league

import "testing"

func matchFixture(participantID, championID int, win bool, puuid string) map[string]interface{} {
	return map[string]interface{}{
		"participantIdentities": []interface{}{
			map[string]interface{}{
				"participantId": float64(participantID),
				"player":        map[string]interface{}{"puuid": puuid, "accountId": float64(111)},
			},
			map[string]interface{}{
				"participantId": float64(participantID%2 + 1), // some other player
				"player":        map[string]interface{}{"puuid": "someone-else", "accountId": float64(222)},
			},
		},
		"participants": []interface{}{
			map[string]interface{}{
				"participantId": float64(participantID),
				"championId":    float64(championID),
				"stats": map[string]interface{}{
					"win": win, "kills": float64(5), "deaths": float64(2), "assists": float64(7),
				},
			},
		},
	}
}

// TestStatsAggregatesAcrossGames checks that Stats() correctly finds "me"
// in each game (by puuid), tallies wins/losses, and groups per champion.
func TestStatsAggregatesAcrossGames(t *testing.T) {
	client := &fakeClient{
		responses: map[string]interface{}{
			"/lol-summoner/v1/current-summoner": map[string]interface{}{
				"puuid": "my-puuid", "accountId": float64(111),
			},
			"/lol-match-history/v1/products/lol/current-summoner/matches?begIndex=0&endIndex=2": map[string]interface{}{
				"games": map[string]interface{}{
					"games": []interface{}{
						matchFixture(1, 103, true, "my-puuid"),  // Ahri, win
						matchFixture(2, 103, false, "my-puuid"), // Ahri, loss
						matchFixture(1, 157, true, "my-puuid"),  // Yasuo, win
					},
				},
			},
			"/lol-game-data/assets/v1/champion-summary.json": []interface{}{
				map[string]interface{}{"id": float64(103), "name": "Ahri"},
				map[string]interface{}{"id": float64(157), "name": "Yasuo"},
			},
		},
	}
	svc := NewService(client)

	result, err := svc.Stats(3)
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}
	if result.Games != 3 {
		t.Fatalf("Games = %d, want 3", result.Games)
	}
	if result.Wins != 2 {
		t.Errorf("Wins = %d, want 2", result.Wins)
	}
	if diff := result.WinRate - 200.0/3; diff > 0.001 || diff < -0.001 {
		t.Errorf("WinRate = %v, want ~%v", result.WinRate, 200.0/3)
	}
	if len(result.PerChampion) != 2 {
		t.Fatalf("PerChampion = %+v, want 2 entries", result.PerChampion)
	}

	var ahri *ChampionStats
	for i := range result.PerChampion {
		if result.PerChampion[i].ChampionID == 103 {
			ahri = &result.PerChampion[i]
		}
	}
	if ahri == nil {
		t.Fatal("no entry for Ahri (103)")
	}
	if ahri.ChampionName != "Ahri" || ahri.Games != 2 || ahri.Wins != 1 {
		t.Errorf("Ahri stats = %+v, want Name=Ahri Games=2 Wins=1", ahri)
	}
}

// TestStatsSkipsGamesWhereLocalPlayerIsntFound guards against a malformed
// or unexpected match entry silently miscounting instead of being skipped.
func TestStatsSkipsGamesWhereLocalPlayerIsntFound(t *testing.T) {
	client := &fakeClient{
		responses: map[string]interface{}{
			"/lol-summoner/v1/current-summoner": map[string]interface{}{"puuid": "my-puuid"},
			"/lol-match-history/v1/products/lol/current-summoner/matches?begIndex=0&endIndex=0": map[string]interface{}{
				"games": map[string]interface{}{
					"games": []interface{}{
						matchFixture(1, 103, true, "not-me"),
					},
				},
			},
		},
	}
	svc := NewService(client)

	result, err := svc.Stats(1)
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}
	if result.Games != 0 {
		t.Errorf("Games = %d, want 0 (local player not found in fixture)", result.Games)
	}
}
