package league

import (
	"sync"
	"time"
)

// Service implements the LoL Remote use cases (champ select, runes,
// ready-check) against a Client port. All state that was previously a
// package-level global lives here instead, so the service is self
// contained and could be instantiated more than once (e.g. in tests with a
// fake Client).
type Service struct {
	client Client

	autoAcceptMu sync.Mutex
	autoAccept   bool

	champsMu    sync.Mutex
	champsCache []Champion

	spellsMu    sync.Mutex
	spellsCache []SummonerSpell

	runeMu      sync.Mutex
	perksCache  map[int]Perk
	stylesCache []Style

	assetMu    sync.Mutex
	assetCache map[string][]byte
}

func NewService(c Client) *Service {
	return &Service{
		client:     c,
		assetCache: map[string][]byte{},
	}
}

func (s *Service) AutoAccept() bool {
	s.autoAcceptMu.Lock()
	defer s.autoAcceptMu.Unlock()
	return s.autoAccept
}

func (s *Service) SetAutoAccept(v bool) {
	s.autoAcceptMu.Lock()
	s.autoAccept = v
	s.autoAcceptMu.Unlock()
}

// Watch auto-accepts ready checks while the option is enabled. Meant to run
// in its own goroutine for the life of the process.
func (s *Service) Watch() {
	for {
		if s.AutoAccept() {
			if phase, err := s.client.Get("/lol-gameflow/v1/gameflow-phase"); err == nil && asString(phase) == "ReadyCheck" {
				_, _ = s.client.Post("/lol-matchmaking/v1/ready-check/accept", nil)
			}
		}
		time.Sleep(time.Second)
	}
}

// Asset proxies and caches a raw LCU asset (champion icon, rune image) for
// the HTTP layer, which has no direct access to the Client.
func (s *Service) Asset(path string) ([]byte, error) {
	s.assetMu.Lock()
	if b, ok := s.assetCache[path]; ok {
		s.assetMu.Unlock()
		return b, nil
	}
	s.assetMu.Unlock()

	b, err := s.client.Raw(path)
	if err != nil {
		return nil, err
	}
	s.assetMu.Lock()
	s.assetCache[path] = b
	s.assetMu.Unlock()
	return b, nil
}

func (s *Service) Accept() error {
	_, err := s.client.Post("/lol-matchmaking/v1/ready-check/accept", nil)
	return err
}

func (s *Service) Decline() error {
	_, err := s.client.Post("/lol-matchmaking/v1/ready-check/decline", nil)
	return err
}

// RawSession returns the unparsed /lol-champ-select/v1/session response.
// Temporary: the LCU's trade/swap schema isn't publicly documented, so this
// is how real field names get confirmed instead of guessed a third time —
// remove once that's settled.
func (s *Service) RawSession() (interface{}, error) {
	return s.client.Get("/lol-champ-select/v1/session")
}

// RawLiveGame returns the unparsed /liveclientdata/allgamedata response.
// Temporary: a CS field assumption (creepScore) is unverified against a
// real client — see parseLiveStats.
func (s *Service) RawLiveGame() (interface{}, error) {
	return s.client.LiveGameData()
}
