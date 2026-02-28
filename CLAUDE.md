# WBS: Preview Scroll + Glow + Chroma Syntax Highlighting

## Tasks
- [x] W1.1 `go get github.com/charmbracelet/glamour` (glow/markdown)
- [x] W1.2 `go get github.com/alecthomas/chroma/v2` (syntax highlight)
- [x] W1.3 Add `GlowPreview bool`, `ChromaPreview bool`, `ChromaStyle string` to BtConfig
- [x] W1.4 Thread new config fields main.go → NewRenderer

- [x] W2.1 Remove height truncation from genPlainTextPreview (return full content)
- [x] W2.2 Add glamour glow rendering for .md / .txt / .markdown (preview_gen.go)
- [x] W2.3 Add chroma syntax highlighting for code files (preview_gen.go)
- [x] W2.4 Update GeneratePreview signature: add glowEnabled, chromaEnabled, chromaStyle params

- [x] W3.1 Add `previewOffset int`, `previewOffsetPath string`, `inFlight map[string]bool` to Renderer
- [x] W3.2 Add `glowEnabled`, `chromaEnabled`, `chromaStyle` to Renderer + NewRenderer params
- [x] W3.3 Fix renderSelectedFileContent: dedup inFlight + apply scroll offset (cropPreviewContent)
- [x] W3.4 Add `cropPreviewContent(content, offset, height)` helper
- [x] W3.5 Add `ScrollPreview(path string, delta int)` method
- [x] W3.6 Update SetPreviewCache to clear inFlight flag

- [x] W4.1 Add `PreviewScrollMsg{Delta int}` to state package
- [x] W4.2 Add J/K (±1 line) and shift+pgdown/shift+pgup (±page) in processKeyDefault
- [x] W4.3 Handle PreviewScrollMsg in model.Update (main.go)
- [x] W4.4 Update help text (render.go)

- [x] W5.1 go build + go test — all pass
- [x] W5.2 bash install.sh → ~/bin/bt
- [x] W5.3 Update key bindings in CLAUDE.md
- [x] W5.4 git commit + push

## Verify
- `bt .` shows syntax-highlighted Go/Python/etc in right pane
- `.md` files rendered with glamour formatting
- J/K scrolls preview 1 line; Shift+PgDn/PgUp scrolls 1 page
- No duplicate goroutines: rapid navigation doesn't pile up preview goroutines
- `go test ./...` all pass

---

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
    theme.go            # ThemeColors, LoadTheme(), DarkThemeColors, LightThemeColors
    preview_gen.go      # Async preview: image (half-block), text, glow(md), chroma(code)
pkg/stack/stack.go      # Generic Stack[T] for DFS traversal
```

### Data Flow
```
KeyMsg → state.ProcessKey() → tree operation / PreviewScrollMsg → model.Update() → ui.Render()
FSEvent → fswatcher → NodeChange → tree.ReadChildren() → re-render
FileSelect → preview_gen goroutine (deduped) → PreviewDoneChan → cache → render
```

### Key Types
- `model` (main.go): BubbleTea model, owns `*state.State` + `*ui.Renderer`
- `state.State`: `Tree`, `OpBuf`, `InputBuf`, `ErrBuf`, `SearchMatches`, `LinearNav`
- `tree.Node`: recursive tree node with `selectedChildIdx`, `showHidden`
- `ui.Renderer`: `previewCache`, `previewOffset`, `inFlight` dedup, `offsetMem` tree scroll

### Operations (state.OpBuf)
`Noop` | `Move(d)` | `Copy(y)` | `Delete(D)` | `Go(gg/G)` | `Insert(i→f/d)` | `Rename(r)` | `SearchInput(/)`

### Config Keys (`~/.config/bt/conf.yaml`)
`padding` (5) | `file_preview` (true) | `highlight_indent` (true) | `in_place_render` (false) | `theme` ("dark") | `linear_nav` (false) | `glow_preview` (true) | `chroma_preview` (true) | `chroma_style` ("friendly")

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
bash install.sh  # build + install to ~/bin/bt
```

## Key Bindings
`j/k` move | `J/K` preview scroll ±1 | `pgdown/pgup` jump 10 | `shift+pgdown/pgup` preview page | `h/l` parent/child | `tab` mark | `d/y` move/copy | `D` delete | `r` rename | `i` insert | `e` edit | `H` hidden | `?` help | `gg/G` top/bottom | `1-9` expand N levels | `0` collapse all | `/` search | `n/N` next/prev match
