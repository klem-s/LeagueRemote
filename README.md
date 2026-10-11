# LoL Remote

Mobile remote for the League of Legends client: accept matches, pick and ban your champion, manage your rune pages, and follow the draft from your phone.

Go rewrite of the original Python script (see "History" below): a single Windows executable, no Python or dependency to install.

## Installation

Download `lol-remote.exe` from the [Releases](../../releases) page and run it (double-click). No console opens: a tray icon appears, along with a window showing the address and PIN to enter on your phone.

Allow the executable through the Windows firewall (private networks) on first launch.

## Architecture

```
Phone (PWA)  ──HTTP/HTTPS──▶  lol-remote.exe (PC, port 53217)  ──HTTPS local──▶  League client (LCU API)
```

The League client exposes an unofficial REST API, the **LCU API**, which only listens on `127.0.0.1`. The phone can't talk to it directly, so `lol-remote.exe` acts as a relay.

### Connecting to the LCU

- The client's port and auth token are read directly from the running `LeagueClientUx.exe` process (command line, via PowerShell), which works regardless of install location.
- As an alternative, the `LOL_REMOTE_LOCKFILE` environment variable can point at the client's `lockfile` instead.
- Basic Auth, user `riot`. The client's certificate is self-signed, so SSL verification is disabled for these requests.
- The port and password change every time the client restarts; a 401/403 or connection error triggers an automatic re-read of the credentials.

## Running it

```
lol-remote.exe
```

A window shows the address, PIN and a QR code on launch; scanning the QR opens the app and logs straight in, no need to retype the PIN. The tray icon lets you reopen this window at any time, or quit.

The PIN is random on every launch unless the `LOL_REMOTE_PIN` environment variable is set. The default port is `53217`; if it's already taken, a free one is chosen automatically.

### Access outside Wi-Fi (optional)

With Tailscale on the PC and phone, `tailscale serve --bg 53217` makes the app reachable at `https://<pc>.<tailnet>.ts.net`, including over mobile data (adjust the port if the app picked a different one).

## Features

**Match and champ select**
- "Match found" screen with Accept/Decline, plus an auto-accept option.
- Champion grid with search. Unplayable or already-taken champions are greyed out.
- Tapping a champion hovers it immediately (no separate button). A separate Lock in / Ban button commits it. Most queues run simultaneous bans, so a player's own ban stays active for the whole ban phase — a tap then targets the ban. A "Preview as my pick" button appears alongside it (while banning, with a pick still pending) as an explicit separate action for broadcasting a pick preview during that window.
- Request, receive, accept or decline a swap with a teammate — champion, position (lane/role) or pick order — via a small icon on their portrait, pulsing when they've sent you a request.
- Phase timer, turns red under 10 seconds.
- Draft overview: ally picks and hovers (hover shown translucent), enemy picks, both teams' bans, roles, and a highlight on whoever is currently picking. Names only show when the client provides them (Riot hides them in ranked).

**In game**
- Once the match itself starts, the app shows a live match clock instead of a stale champ-select screen.

**Runes**
- Page list, one-tap activation, delete editable pages.
- Full editor: primary tree and keystone, 3 primary rows, secondary tree (2 runes on 2 different rows), 3 shard rows.
- Client-recommended runes for the chosen champion and assigned role. "Apply" overwrites the active page if it's editable, otherwise creates a new one.

**Web app**
- Manifest and icon generated as PNG on launch: the app can be added to the home screen and opens full-screen.

**On the PC**
- No console: a tray icon, "Show" menu item (address, PIN and QR in a window matching the mobile app's hextech theme) and "Quit".
- If the WebView2 runtime isn't installed (rare — present by default on Windows 11 and deployed via Windows Update on most Windows 10 machines), falls back automatically to a system dialog.
- Single instance only: relaunching the executable while it's already running just shows a message instead of opening a second icon/server.
- Checks for a newer version on launch (GitHub releases); if one exists, a button in the window links to it. No auto-download or replacement.

## Project structure

Layered (clean architecture):

| Package | Role |
|---|---|
| `internal/league` | Use cases: `Service`, data models, the `Client` interface (the port), business errors. Depends on nothing else in this project. |
| `internal/lcu` | Adapter: implements `league.Client` on top of the real LCU API (HTTPS, credential discovery). |
| `internal/icon` | PNG/ICO icon and manifest generation, no dependency. |
| `internal/httpapi` | HTTP adapter: routes, JSON, PIN check, embedded mobile page (`page.html`). Depends only on `league` and `icon`. |
| `internal/trayui` | Tray icon and info window (`popup.html`, Windows only, `//go:build windows`); a console-only fallback compiles on other OSes for development. |
| `internal/singleinstance` | Named Windows mutex preventing two instances from running at once. |
| `internal/updatecheck` | Calls the GitHub API (`/releases/latest`) to check for a newer version. |
| `main.go` | Composition root: wires `lcu.Client` → `league.Service` → `httpapi.Server` → `trayui`. |

### Server routes

All `/api/*` routes require the `X-Pin` header.

| Method | Route | Role |
|---|---|---|
| GET | `/` | mobile page |
| GET | `/manifest.webmanifest`, `/icon-192.png`, `/icon-512.png` | web app |
| GET | `/icon/<id>` | a champion's icon (LCU proxy, cached) |
| GET | `/asset?p=/lol-game-data/assets/...` | proxy for client images (runes, trees) |
| GET | `/api/state` | phase, select state, player's pending action, teams |
| GET | `/api/champions` | `{id, name}` list |
| GET | `/api/runes` | pages, active page, trees and runes |
| GET | `/api/runes/recommended` | recommended runes for the current champion |
| POST | `/api/select` | `{actionId, championId, lock}`: hovers, then locks in if `lock`. Also used to pre-pick before your turn (see below) — the same endpoint, just allowed earlier |
| POST | `/api/swaps/request`, `/accept`, `/decline`, `/cancel` | `{type, id}`: drive a champion/position/pick-order swap with a teammate (see below) |
| POST | `/api/accept`, `/api/decline` | ready check |
| POST | `/api/auto-accept` | `{enabled}` |
| POST | `/api/runes/activate`, `/api/runes/delete` | `{id}` |
| POST | `/api/runes/save` | `{id?, name, primaryStyleId, subStyleId, selectedPerkIds[9]}` |
| POST | `/api/runes/apply` | same fields, onto the active page or a new one |

## Building from source

```
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H=windowsgui -X main.version=v0.0.0-dev" -o lol-remote.exe .
```

`-X main.version=...` embeds the version the update check compares against GitHub's latest release; the release CI sets it from the pushed tag automatically. Omit it (or leave it as the default `dev`) for local builds — the update check just skips itself.

`-H=windowsgui` keeps a console from opening behind the tray icon. Three dependencies (none use cgo, so cross-compilation from Linux/macOS works):
- [`getlantern/systray`](https://github.com/getlantern/systray) for the tray icon.
- [`jchv/go-webview2`](https://github.com/jchv/go-webview2) for the window (pure-Go binding to Windows' WebView2 control).
- [`skip2/go-qrcode`](https://github.com/skip2/go-qrcode) to generate the QR code embedded in that window.

## Publishing a release

Every `vX.Y.Z` tag pushed to GitHub triggers CI (`.github/workflows/release.yml`), which builds `lol-remote.exe` for Windows and publishes a Release with the binary attached.

```
git tag v1.0.0
git push origin v1.0.0
```

## Things to watch for

- The LCU isn't an official API: endpoints can change after a patch.
- The in-game clock uses a *different* local server than the LCU: the Live Client Data API at `127.0.0.1:2999` (unauthenticated, only reachable while a match is actually running), not `127.0.0.1:<lcu-port>`. Unlike the rest of this app, that endpoint shape is officially documented by Riot, so it's high-confidence.
- `champion-summary.json` contains special-mode variants sharing a name with a different ID; the smallest ID per name is kept.
- During a ban, the allowed-champion list is computed server-side (everything except already-banned or locked-in champions), since `bannable-champion-ids` isn't reliable.
- On some client versions, editing a page (`PUT`) fails: the page is then deleted and recreated.
- Over `http` on Wi-Fi, the PIN travels in the clear on the local network. Don't expose the port directly to the Internet.
- Pre-picking before your turn: the actual bug (after three wrong guesses at LCU-side restrictions — `my-selection`, `timer.phase == "PLANNING"`, "bans resolved") turned out to be on this side. `myAction()` returns whichever of the player's own actions is `isInProgress`, prioritizing that over anything else — so while a player's own ban is active, it returns the ban, not their separately-pending pick, and the mobile page was PATCHing the hovered champion onto the *ban* action. `SelectState.UpcomingPick` (`myPickAction()` in `champselect.go`) now exposes the player's own pick action independently, and Hover/Preview targets that instead of `Action` — see `TestStateUpcomingPickDuringBan`.
- Swaps come in three kinds, each a separate array in the session — `trades` (champion), `positionSwaps` (lane/role), `pickOrderSwaps` — all shaped `{id, cellId, state}` with `state` one of `AVAILABLE`/`SENT`/`RECEIVED`/`BUSY`/`INVALID`. This part is confirmed against a real session dump (`internal/league/champselect_test.go`'s `TestBuildTeamsSwapsRealSession`), not guessed. The *action* URLs (`.../trades/{id}/request`, `.../position-swaps/{id}/request`, `.../pick-order-swaps/{id}/request`, each with `/accept`, `/decline`, `/cancel`) follow the LCU's usual kebab-case convention and the `trades` ones are corroborated by several independent sources, but `position-swaps`/`pick-order-swaps` are an educated extrapolation, not yet exercised against a live client — if request/accept/decline/cancel misbehaves for those two, the LCU's actual error message (surfaced as a toast) is the fastest way to fix it.

## History

This project started as a single Python script (`lol_remote.py`, standard library only), to spare every player from installing Python. The Go rewrite produces a native executable instead.

---

LoL Remote isn't endorsed by Riot Games and doesn't reflect the views or opinions of Riot Games or anyone officially involved in producing or managing League of Legends. League of Legends and Riot Games are trademarks or registered trademarks of Riot Games, Inc. League of Legends © Riot Games, Inc.
