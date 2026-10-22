# charm-logs

A log viewer component for [Bubble Tea](https://github.com/charmbracelet/bubbletea): circular buffer, follow mode, level and text filters, plain-text export, and a `log/slog` bridge.

> **[Leia em Português (PT-BR)](README.pt-br.md)**

## Install

```bash
go get github.com/junhinhow/charm-logs@latest
```

## Usage

```go
package main

import (
	"log"
	"log/slog"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/junhinhow/charm-logs"
)

type model struct {
	logs logs.Model
}

func newModel() model {
	viewer := logs.New(logs.WithMaxEntries(500))
	viewer.SetSize(80, 20)

	logger := slog.New(viewer.SlogHandler())
	logger.Info("application started", "version", "1.0")

	viewer.Append(logs.Entry{
		Time:    time.Now(),
		Level:   logs.Info,
		Source:  "download",
		Message: "downloading file",
	})

	return model{logs: viewer}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.logs.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	updated, cmd := m.logs.Update(msg)
	m.logs = updated.(logs.Model)
	return m, cmd
}

func (m model) View() tea.View { return m.logs.View() }

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		log.Fatal(err)
	}
}
```

Without an initial `SetSize`, the viewer uses 80x24. Handling `tea.WindowSizeMsg`
keeps it in sync with terminal resizes.

## Keys

| Key | Action |
|-----|--------|
| `j` / `↓` | Select the next (newer) entry |
| `k` / `↑` | Select the previous (older) entry; turns follow off |
| `g` | Go to the oldest entry |
| `G` | Go to the newest entry and turn follow on |
| `/` | Enter search mode |
| `enter` | Confirm the search |
| `esc` | Cancel the search and restore the previous filter |
| `backspace` | Delete the last character while searching |
| `l` | Cycle the minimum level: debug → info → warn → error → debug |
| `f` | Toggle follow |
| `c` | Clear the buffer |
| `e` | Export the visible entries to `logs.txt` in the current directory |

While search mode is active, printable keys are text: `j`, `k`, `g`, `l` and
the other shortcuts are not interpreted as navigation.

## API

| Symbol | Description |
|--------|-------------|
| `Level`, `Debug`, `Info`, `Warn`, `Error` | Severity, ordered from least to most severe |
| `Level.String`, `ParseLevel` | Text conversion; accepts `debug`, `info`, `warn`, `warning`, `error` |
| `Entry` | `Time`, `Level`, `Source` and `Message` of one line |
| `New(opts ...Option)` | Creates the viewer (defaults: 1000 entries, Debug, follow on, 80x24) |
| `WithMaxEntries(n)` | Sets the circular buffer capacity; values below 10 are raised to 10 |
| `Model.Append(entries ...Entry)` | Appends entries and returns a command that emits `AppendedMsg` |
| `Model.Appendf(level, format, args...)` | Formats a message with the current time and appends it |
| `Model.SetSize(width, height)` | Sets the viewport size |
| `Model.SetMinLevel(level)` | Hides entries below the level during rendering only |
| `Model.SetFollow(bool)`, `Model.Follow()` | Controls and reads follow mode |
| `Model.SetFilter(string)`, `Model.Filter()` | Controls and reads the text filter |
| `Model.Entries()` | Copy of the visible entries (after level and text filters) |
| `Model.Len()` | Total entries in the buffer, including hidden ones |
| `Model.Clear()` | Empties the buffer |
| `Model.Export(io.Writer)` | Writes visible entries as plain text, without ANSI |
| `Model.SlogHandler()` | Returns a concurrency-safe `slog.Handler` backed by the buffer |
| `Model.Init`, `Model.Update`, `Model.View` | `tea.Model` implementation |
| `AppendedMsg`, `ExportedMsg` | Messages produced by the append and export commands |

## Design notes

- **Circular buffer.** The default capacity is 1000. When it is full, the
  oldest entry is discarded; memory usage does not grow with the log stream.
- **Follow.** Follow is on by default. Appending new entries keeps the newest
  entry selected. Pressing `k`/`↑` turns follow off; `G` and `f` turn it back
  on.
- **Stable selection.** The selected entry is tracked by an internal sequence
  number rather than a slice index, so new entries arriving do not move the
  selection while follow is off, and entries evicted from the buffer do not
  leave a dangling index.
- **Filtering.** The text filter is a case-insensitive substring match against
  both `Message` and `Source`. `SetMinLevel` affects rendering only; entries
  remain in the buffer and reappear if the level is lowered again.
- **Layout.** Each visible line is `HH:MM:SS LEVEL [source] message`. ANSI
  escapes and control characters in messages are neutralized, and every line
  is truncated with `ansi.Truncate`, so `View` never exceeds the configured
  width. The view never exceeds the configured height either.
- **Export.** `Export` writes RFC 3339 timestamps and uppercase level names:
  `2006-01-02T15:04:05Z07:00 LEVEL [source] message`. Messages are flattened
  to a single line. The `e` key uses the same format to write `logs.txt`.
- **slog.** `SlogHandler` maps slog levels to `Debug`, `Info`, `Warn` and
  `Error`, appends attributes as `key=value`, and expands groups with dots
  (`req.user.name=ana`). `Handle` accepts every level; use `SetMinLevel` to
  hide entries in the UI.

## Limitations

- Messages are rendered on one line; there is no wrapping or expandable entry.
- No mouse support, selection of arbitrary ranges, or clipboard integration.
- The color palette is fixed and not customizable.
- `SlogHandler` does not trigger a Bubble Tea repaint by itself. If records
  arrive outside the `Update` loop, send a message to the program (or return
  `Model.Append`'s command from `Update`) so the view refreshes.
- `Append` and `SlogHandler` are safe for concurrent use because only the
  buffer is protected by a mutex. Methods that change UI state (`SetSize`,
  `SetFilter`, `SetMinLevel`, `Clear`, `Update`, ...) must be called from the
  Bubble Tea update loop.

## License

MIT
