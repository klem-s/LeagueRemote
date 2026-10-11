package league

import "fmt"

// Client is the port this package needs to talk to a League client: it
// knows nothing about HTTP, TLS, lockfiles or PowerShell. An adapter
// (internal/lcu) implements it; the service only depends on this interface,
// never on the adapter.
type Client interface {
	Get(path string) (interface{}, error)
	Post(path string, body interface{}) (interface{}, error)
	Put(path string, body interface{}) (interface{}, error)
	Patch(path string, body interface{}) (interface{}, error)
	Delete(path string) (interface{}, error)
	Raw(path string) ([]byte, error)

	// LiveGameData returns the raw /liveclientdata/allgamedata response
	// (game time, and every player's live champion/level/KDA/CS) via the
	// separate, unauthenticated Live Client Data API the running game
	// process exposes on 127.0.0.1:2999 — a different local server than
	// the LCU, only up while a match is actually in progress.
	LiveGameData() (interface{}, error)
}

// RemoteError is returned by a Client when the League client answered with
// a non-2xx status (as opposed to being unreachable). Its Body often
// carries a JSON {"message": "..."} the caller can surface to the user.
type RemoteError struct {
	Code int
	Body []byte
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("HTTP %d", e.Code)
}

// ValidationError signals a business-rule failure (not a transport
// problem) that should be reported to the user as-is, e.g. "this page
// can't be edited".
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }
