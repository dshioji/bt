# bt - Better Tree File Manager

## Architecture

**Module**: `github.com/LeperGnome/bt` | **Go**: 1.24.4 | **Framework**: Bubble Tea (TUI)

### Package Structure
```
cmd/bt/main.go          # Entry point, BubbleTea model (Init/Update/View)
internal/
  config/config.go      # Viper YAML config (~/.config/bt/conf.yaml) + CLI flags
  state/state.go        # App state, keyboard routing, operation mode (OpBuf)
  tree/
    tree.go             # File ops: nav, copy/move/delete/rename, mark
    node.go             # Node struct (Path, Info, Children, Parent)
    fswatcher.go        # fsnotify watcher → NodeChange channel
  ui/
    render.go           # Renderer: tree (left 50%) + preview/help (right 50%)
    stylesheets.go      # lipgloss styles, color scheme
    preview_gen.go      # Async preview: image (half-block Unicode) + text
pkg/stack/stack.go      # Generic Stack[T] for DFS traversal
```

### Data Flow
```
KeyMsg → state.ProcessKey() → tree operation → model.Update() → ui.Render()
FSEvent → fswatcher → NodeChange → tree.ReadChildren() → re-render
FileSelect → preview_gen goroutine → PreviewDoneChan → cache → render
```

### Key Types
- `model` (main.go): BubbleTea model, owns `*state.State` + `*ui.Renderer`
- `state.State`: `Tree`, `OpBuf` (current op), `InputBuf`, `ErrBuf`
- `tree.Node`: recursive tree node with `selectedChildIdx`, `showHidden`
- `ui.Renderer`: `previewCache map[string]Preview`, `offsetMem` for scroll

### Operations (state.OpBuf)
`Noop` | `Move(d)` | `Copy(y)` | `Delete(D)` | `Go(gg/G)` | `Insert(i→f/d)` | `Rename(r)`

### Config Keys (`~/.config/bt/conf.yaml`)
`padding` (5) | `file_preview` (true) | `highlight_indent` (true) | `in_place_render` (false) | `theme` ("dark")

### Theme System (`internal/ui/theme.go`)
- Built-ins: `dark` (default), `light`
- Custom: set `theme: mytheme` → loads `~/.config/bt/mytheme.yaml`
- Partial files ok — missing keys fall back to dark defaults
- See `examples/light.yaml` for all color keys
- `ThemeColors.ToStylesheet()` builds `Stylesheet` from hex strings

## Commands
```bash
make build    # → ./bin/bt
make test     # go test -v ./...
make install  # → ~/.local/bin/bt
make lint     # go fmt
```

## Key Bindings
`j/k` move | `pgdown/pgup` jump 10 | `h/l` parent/child | `tab` mark | `d/y` move/copy | `D` delete | `r` rename | `i` insert | `e` edit | `H` hidden | `?` help | `gg/G` top/bottom | `1-9` expand N levels | `0` collapse all | `/` search | `n/N` next/prev match
