# sundial: Design Spec

- **Date:** 2026-10-01
- **Status:** Approved design, pending spec review
- **Scope:** v1 (read-only)

## 1. Summary

sundial is a terminal calendar for macOS. It reads events from the system calendar store (EventKit), which covers every account macOS syncs (iCloud, Google, Exchange, local), and presents them in month, week, and agenda views built with the Charm stack. A command palette gives fuzzy access to every action. Settings live in `~/.config/sundial/config.toml`.

## 2. Goals and Non-Goals

### Goals (v1)

- Read-only browsing of all macOS calendars.
- Three views: month, week, agenda, plus an event detail pane.
- Command palette (`ctrl+k` / `:`) with fuzzy matching.
- Support both quick-glance use (fast startup) and always-open use (live refresh when calendars change, moving "now" line, midnight rollover).
- Non-interactive CLI subcommands (`today`, `agenda`, `calendars`) for scripts and shell greetings.
- TOML config with sensible built-in defaults; works with no config file.
- Pastel default theme that makes headers and labels stand out.
- Public-repo safe: no real calendar data, personal paths, or secrets ever committed.

### Non-Goals (v1)

- Creating, editing, or deleting events (planned later; the architecture reserves room for it, see section 4.2).
- Desktop notifications or reminders.
- Non-macOS platforms.
- Reminders (EKReminder) support.
- Release automation (goreleaser, Homebrew tap) — `just install` is sufficient for v1.

## 3. Tech Stack

- Go 1.27, cgo with an Objective-C bridge to EventKit.
- macOS 14+ (uses `requestFullAccessToEventsWithCompletion:`).
- Bubble Tea v2, Lip Gloss v2, Bubbles v2.
- `spf13/cobra` + `charmbracelet/fang` for the CLI.
- `pelletier/go-toml/v2` for config.
- `sahilm/fuzzy` for palette matching.
- `charmbracelet/x/exp/teatest` for TUI golden tests.
- `golangci-lint` pinned as a Go tool dependency in `go.mod` (`go tool golangci-lint`).
- `just` as the task runner.

## 4. Architecture

### 4.1 Package Layout

```
sundial/
  cmd/sundial/           main: cobra + fang wiring
  internal/
    calendar/            domain types (Event, Calendar, Range) + Source interface
    eventkit/            cgo + Objective-C EventKit Source (darwin build tag)
    fakesource/          in-memory Source with fictional events (tests, demos)
    store/               window cache over a Source; refetch on change
    config/              TOML load, defaults, validation, XDG path resolution
    tui/
      app.go             root model: key routing, view switching, palette overlay
      month/             month view model
      week/              week view model
      agenda/            agenda view model
      detail/            event detail pane
      palette/           command palette + command registry
      theme/             Lip Gloss styles built from config
    cli/                 non-interactive subcommands (text / JSON output)
  justfile
  Info.plist             embedded into the binary for TCC usage description
```

### 4.2 Boundaries

- Views never touch EventKit. They depend only on `store`, which depends only on `calendar.Source`.
- `calendar.Source` (read-only):

  ```go
  type Source interface {
      Calendars(ctx context.Context) ([]Calendar, error)
      Events(ctx context.Context, r Range) ([]Event, error)
      Watch(ctx context.Context) (<-chan struct{}, error)
  }
  ```

- Future writes go in a separate `Writer` interface (`Create`, `Update`, `Delete`) that a Source may also implement. Readers are unaffected when it is added. Write commands will register in the same palette registry.

### 4.3 Domain Types

- `Calendar`: `ID`, `Title`, `Color` (hex), `Source` (account name), `ReadOnly`.
- `Event`: `ID`, `CalendarID`, `Title`, `Start`, `End`, `AllDay`, `Location`, `Notes`, `URL`, `Attendees []string` (display names only), `Status`.
- `Range`: `Start`, `End` (half-open, local time).

### 4.4 Data Flow

1. The root model asks `store` for the visible range (plus a buffer around it).
2. `store` returns cached events or fetches from `Source` asynchronously via a `tea.Cmd`, delivering an `eventsLoadedMsg`.
3. The active view renders from the data it was given.
4. `Source.Watch` events become a `refreshMsg`; `store` invalidates and refetches the visible window.
5. A 1-minute `tea.Tick` updates the "now" line and handles midnight rollover.

## 5. TUI

### 5.1 Global Keys

| Key | Action |
|---|---|
| `m` / `w` / `a` | Month / week / agenda view |
| `t` | Jump to today |
| `h`/`l`, `←`/`→` | Previous / next period |
| `j`/`k`, `↑`/`↓` | Move within view |
| `enter` | Open event detail |
| `esc` | Close detail / palette |
| `ctrl+k`, `:` | Command palette |
| `?` | Help (short / full) |
| `q`, `ctrl+c` | Quit |

All bindings are overridable via `[keys]` in config.

### 5.2 Month View

- 7-column, 5–6 row grid; week starts per `week_start`.
- Each cell shows the day number and as many events as fit, followed by `+N more`.
- Event color comes from its calendar's color (pastelized, see 5.7).
- Today and the selected day are highlighted.

### 5.3 Week View

- 7 day columns over an hour grid; all-day events in a strip at the top.
- A "now" line marks the current time.
- Vertically scrollable; initial scroll position is `day_start`.
- If the terminal is too narrow for 7 columns, fall back to a 3-day view.

### 5.4 Agenda View

- Scrollable list grouped by day headers, starting at today.
- Loads more days as you scroll.
- Rows: time range, colored calendar dot, title, location.
- `/` filters (bubbles/list).

### 5.5 Detail Pane

Shows title, time, calendar, location, notes, attendees, and URL. `o` opens the meeting/event URL with `open`.

### 5.6 Status Bar

View name, current date range, sync indicator, key hints. Errors display here in the warning color.

### 5.7 Theme

Default pastel palette, with light and dark variants via Lip Gloss `LightDark`:

| Role | Name | Hex (dark) |
|---|---|---|
| Header / view title | lavender | `#C6A0F6` |
| Day headers | sky | `#91D7E3` |
| Today highlight | peach | `#F5A97F` |
| Selection | pink | `#F5BDE6` |
| Now-line / accents | mint | `#A6DA95` |
| Muted text | overlay | `#8087A2` |
| Warnings / sync errors | rose | `#ED8796` |

Light-mode variants are darker tints of the same hues, chosen during implementation to keep contrast readable on light backgrounds.

`pastelize = true` (default) blends each macOS calendar color 35% toward white so it harmonizes with the theme. Every palette role is overridable in `[theme]`.

## 6. Command Palette

- Opened with `ctrl+k` or `:`; rendered as a centered overlay.
- Text input with fuzzy-matched results (`sahilm/fuzzy`).
- Single command registry. Each entry: `Name`, `Description`, `Keys`, `Action func(*Model) tea.Cmd`. The help view is generated from the same registry, so they cannot drift.
- v1 commands:
  - Go to month / week / agenda view
  - Go to today
  - Go to date (accepts `2026-10-15`, `next fri`, `+3w`, `-2d`)
  - Toggle calendar visibility (one entry per calendar)
  - Search events by title (jumps to the selected match)
  - Reload
  - Open config in `$EDITOR`
  - Quit
- Typing something that parses as a date offers "Go to <date>" as the top result.

## 7. Configuration

- Path: `$XDG_CONFIG_HOME/sundial/config.toml`, falling back to `~/.config/sundial/config.toml`.
- A missing file means built-in defaults are used. `sundial config init` writes a commented starter file (refuses to overwrite an existing one).
- Invalid config (bad TOML, unknown enum value, bad color) produces an error naming the key and line, and sundial exits nonzero. Unknown keys are reported as errors, not silently ignored.

```toml
default_view = "agenda"     # month | week | agenda
week_start   = "sunday"     # sunday | monday
time_format  = "12h"        # 12h | 24h
day_start    = 8            # week view initial scroll hour (0-23)

[calendars]
hidden = []                 # calendar titles or IDs to hide

[agenda]
days = 14                   # default lookahead (CLI and initial list)

[theme]
pastelize = true
# header    = "#C6A0F6"
# day       = "#91D7E3"
# today     = "#F5A97F"
# selection = "#F5BDE6"
# accent    = "#A6DA95"
# muted     = "#8087A2"
# warning   = "#ED8796"

[keys]
# palette = ["ctrl+k", ":"]
```

## 8. EventKit Bridge

### 8.1 C API

`internal/eventkit/bridge.m` exposes:

- `sd_auth_status()`: returns the current authorization status.
- `sd_request_access()`: blocks until the user responds; returns granted / denied.
- `sd_calendars()`: returns a JSON C string.
- `sd_events(double start_unix, double end_unix)`: returns a JSON C string, with recurrences expanded via `predicateForEventsWithStartDate:endDate:calendars:`.
- `sd_watch_start()` / `sd_watch_stop()`: register or unregister an `EKEventStoreChangedNotification` observer on a background `NSOperationQueue`. The observer calls an exported Go function (`sundialStoreChanged`).
- `sd_free(char*)`.

Go unmarshals the JSON and frees the string. JSON keeps the cgo surface thin; the overhead is negligible at expected volumes (hundreds of events per window).

### 8.2 Watch

The exported Go callback does a non-blocking send on an internal channel. `Watch` debounces notifications by 500 ms before emitting on the returned channel.

### 8.3 Permissions

- `Info.plist` containing `NSCalendarsFullAccessUsageDescription` (and a bundle identifier) is embedded via `-ldflags "-extldflags '-sectcreate __TEXT __info_plist Info.plist'"`. The `justfile` passes these flags.
- macOS attributes the prompt to the hosting terminal app (expected CLI behavior).
- Status `notDetermined`: request access on startup.
- Status `denied` / `restricted`: the TUI shows a full-screen explanation (System Settings → Privacy & Security → Calendars) with `o` to open `x-apple.systempreferences:com.apple.preference.security?Privacy_Calendars`. CLI commands print the same message to stderr and exit with code 2.

### 8.4 Feasibility Spike (first implementation task)

Before building on the bridge, a throwaway spike (about 1 hour) confirms on macOS 26:

1. A cgo binary with the embedded `Info.plist` triggers the permission prompt from a terminal and can read events.
2. `EKEventStoreChangedNotification` fires on a background queue without a main run loop.

If either fails, stop and revise this section before continuing.

## 9. Error Handling

- Fetch failure: keep the last good data on screen, show the error in the status bar, and retry on the next change notification or `reload`.
- Time: all display is in the local time zone. All-day events are treated as dates so they never shift across zones.
- Panics in the TUI restore the terminal (Bubble Tea default) and print the error.

## 10. CLI

| Command | Behavior |
|---|---|
| `sundial` | Launch TUI in `default_view` |
| `sundial today` | Today's agenda; styled when stdout is a TTY, plain when piped |
| `sundial agenda [--days N] [--from DATE] [--json]` | Agenda for a range |
| `sundial calendars [--json]` | List calendar titles and IDs |
| `sundial config init \| path \| edit` | Manage the config file |

Global flags: `--config PATH`, `--debug`, `--debug-verbose`. Hidden: `--source fake` (also `SUNDIAL_SOURCE=fake`).

Exit codes: `0` ok, `1` general error, `2` calendar access denied.

## 11. Privacy and Public-Repo Hygiene

- No telemetry, no network calls.
- No logging unless `--debug`, which writes to `$XDG_STATE_HOME/sundial/debug.log` (fallback `~/.local/state/sundial/debug.log`) and omits event titles, notes, locations, and attendees unless `--debug-verbose`.
- All test fixtures and demo data are fictional.
- Integration tests print counts only, never event content.
- `.gitignore` adds `bin/`, `dist/`, `*.log`, `.env*`, `config.toml` (repo root copies), and VHS output except committed demo assets.
- Every diff is checked for personal data before commit.

## 12. Testing

- **Unit:** `config` (defaults, validation, unknown keys), `store` (caching, invalidation), date math (month grid, week start, DST transitions, all-day handling), palette fuzzy ranking, date-expression parsing.
- **TUI golden:** `teatest` against `fakesource` with a fixed clock and fixed terminal size, covering month, week, agenda, detail, palette open, and access-denied screens.
- **Integration:** `//go:build integration` tests for `eventkit`, run locally only (`just test-integration`), asserting on counts and shapes.
- All tests except integration run without calendar access.

## 13. Tooling

### 13.1 justfile

```
just build             # go build with Info.plist ldflags -> bin/sundial
just run *args         # build + run
just demo              # TUI with fakesource (safe for screenshots / VHS)
just test              # go test ./...
just test-integration  # go test -tags integration ./internal/eventkit/...
just lint              # go tool golangci-lint run
just fmt               # gofmt + goimports
just install           # go install with ldflags into $GOBIN
just clean
```

### 13.2 CI

GitHub Actions on `macos-latest`: lint, then test, then build. No secrets.

## 14. Future Work (out of scope for v1)

- `Writer` interface: quick-add, delete, and reschedule from the palette.
- Desktop notifications before events.
- goreleaser + Homebrew tap.
- Reminders integration.
