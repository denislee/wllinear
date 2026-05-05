# wllinear

A native Wayland GUI client for [Linear.app](https://linear.app), built in Go
with [Gio](https://gioui.org). Mirrors the feature set of
[lazylinear](../lazylinear) but as a graphical desktop app.

## Features

- Browse teams and issues from your Linear workspace
- Sidebar with predefined and dynamic project filters; live counts per filter
- Full and compact issue list views
- Issue detail view with status, priority, assignee, project, labels, dates
- Create issue (title / description / state / assignee / project / priority)
- Edit issue (same fields as create)
- Change workflow status
- Issue search across your assigned issues (`Ctrl+K`)
- One-click "open in browser" for any issue
- Clipboard export of last-cycle issues per project (sidebar shortcut)
- Auto-labeling of unlabeled issues via Gemini CLI (matches lazylinear's flow)
- Persistent UI state: last team, last filter, compact mode

## Requirements

- Linux with Wayland (also runs on X11 — Gio falls back automatically)
- Go 1.25+
- A Linear API key — get one at https://linear.app/settings/api
- Optional: `gemini` CLI in `$PATH` for the auto-label feature

## Configuration

Set the API key one of three ways (priority order):

1. `WLLINEAR_API_KEY` environment variable
2. `LAZYLINEAR_API_KEY` environment variable (fallback for shared setups)
3. `~/.config/wllinear/config.yaml`:

   ```yaml
   api_key: lin_api_xxx
   default_team: ENG
   ```

   `~/.config/lazylinear/config.yaml` is also picked up if no wllinear config
   is present.

UI state (last team, last filter, compact mode) is persisted to
`~/.config/wllinear/state.json`.

## Build & run

```bash
go build -o wllinear .
./wllinear
```

Or via the Makefile:

```bash
make build   # build the binary
make run     # build + run
make tidy    # go mod tidy
make fmt     # go fmt ./...
```

## Keyboard shortcuts

| Scope | Key | Action |
| --- | --- | --- |
| Global | `tab` / `shift+tab` | Switch panel focus |
| Global | `c` | Create issue |
| Global | `ctrl+k` | Search my issues |
| Global | `v` | Toggle compact list mode |
| Global | `?` | Toggle help overlay |
| Global | `q` / `ctrl+c` | Quit |
| Sidebar | `j` / `k` / arrows | Move highlight |
| Sidebar | `enter` / `l` | Apply filter / focus issues |
| Issue list | `j` / `k` / arrows | Navigate |
| Issue list | `enter` | Open in browser |
| Issue list | `l` | Open issue detail |
| Issue list | `e` | Edit issue |
| Issue list | `s` | Change status |
| Issue list | `r` | Refresh |
| Issue list | `t` | Auto-label (only in *My Unlabeled Issues*) |
| Detail | `esc` / `h` | Back to list |
| Modal | `esc` | Cancel |
| Modal | `enter` | Submit |

Mouse-driven equivalents are available everywhere — click on filters, teams,
list rows, modal options, etc.

## Architecture

```
main.go                        program entry; sets up the Gio window
internal/
  config/                      API key + persisted UI state
  linear/                      Linear GraphQL client (queries, mutations, types)
  ai/                          Gemini-CLI wrapper for auto-labeling
  ui/                          Theme: colors, fonts, derived helpers
  app/
    state.go                   Central State + event channel
    events.go                  Event types and their apply() reducers
    commands.go                Async API/IO functions invoked from goroutines
    modals.go                  Modal data structures (create/edit/status/search)
    app.go                     Run loop, key handling, layout dispatch
    sidebar.go                 Sidebar layout (teams + filters + projects)
    main_panel.go              Issue list + issue detail views
    modals_layout.go           Modal overlays
```

State is mutated only on the UI goroutine. Async work runs in goroutines that
push `Event` values through `State.Events`; `State.Wakeup` (bound to
`Window.Invalidate`) requests a redraw, and the next frame applies any pending
events before laying out.

The `internal/linear` and `internal/ai` packages are direct ports from
lazylinear so the GraphQL surface stays identical.

## License

MIT
