package state

import (
	"os"
	"os/exec"
	runtime "runtime"

	t "github.com/LeperGnome/bt/internal/tree"
	tea "github.com/charmbracelet/bubbletea"
)

// PreviewScrollMsg is emitted when the user scrolls the preview pane.
// Handled by model.Update which calls renderer.ScrollPreview.
type PreviewScrollMsg struct{ Delta int }

type Operation int

const (
	Noop Operation = iota
	Move
	Copy
	Delete
	Go
	Insert
	InsertFile
	InsertDir
	Rename
	SearchInput
)

func (o Operation) Repr() string {
	return []string{
		"",
		"moving",
		"copying",
		"confirm removing (y/n) of",
		"g",
		"create new (f)ile/(d)irectory",
		"enter new file name:",
		"enter new directory name:",
		"renaming",
		"/",
	}[o]
}
func (o Operation) IsInput() bool {
	switch o {
	case InsertDir, InsertFile, Rename, SearchInput:
		return true
	default:
		return false
	}
}

type State struct {
	Tree        *t.Tree
	OpBuf       Operation
	PrevOpBuf   Operation
	InputBuf    []rune
	ErrBuf      string
	NodeChanges <-chan t.NodeChange
	HelpToggle  bool

	SearchMatches []*t.Node
	SearchIdx     int
	SearchQuery   string

	LinearNav bool
}

func InitState(root string, linearNav bool) (*State, error) {
	tree, ncc, err := t.InitTree(root, nil)
	if err != nil {
		return nil, err
	}
	return &State{
		Tree:        tree,
		OpBuf:       Noop,
		InputBuf:    []rune{},
		NodeChanges: ncc,
		LinearNav:   linearNav,
	}, nil
}

func (s *State) ProcessNodeChange(nodeChange t.NodeChange) tea.Cmd {
	err := s.Tree.RefreshNodeParentByPath(nodeChange.Path)
	if err != nil {
		s.ErrBuf = err.Error()
	}
	s.Tree.RemoveNodeFromMarkByPath(nodeChange.Path)
	return nil
}

func (s *State) ProcessKey(msg tea.KeyMsg) tea.Cmd {
	switch s.OpBuf {
	case Noop:
		return s.processKeyDefault(msg)
	case Move:
		return s.processKeyMove(msg)
	case Delete:
		return s.processKeyDelete(msg)
	case Copy:
		return s.processKeyCopy(msg)
	case Go:
		return s.processKeyGo(msg)
	case Insert:
		return s.processKeyInsert(msg)
	case InsertFile:
		return s.processKeyInsertFile(msg)
	case InsertDir:
		return s.processKeyInsertDir(msg)
	case Rename:
		return s.processKeyRename(msg)
	case SearchInput:
		return s.processKeySearch(msg)
	default:
		return s.processKeyDefault(msg)
	}
}
func (s *State) processKeyRename(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.RenameMarked(string(s.InputBuf))
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.Tree.DropMark()
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyInsert(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "f":
		s.OpBuf = InsertFile
	case "d":
		s.OpBuf = InsertDir
	default:
		s.OpBuf = Noop
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyInsertFile(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.CreateFileInCurrent(string(s.InputBuf))
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyInsertDir(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		err := s.Tree.CreateDirectoryInCurrent(string(s.InputBuf))
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.OpBuf = Noop
		s.InputBuf = []rune{}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}
func (s *State) processKeyAnyInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "esc":
		s.OpBuf = Noop
		s.PrevOpBuf = Noop
		s.InputBuf = []rune{}
		s.Tree.DropMark()
	case "backspace":
		if l := len(s.InputBuf); l > 0 {
			s.InputBuf = s.InputBuf[:l-1]
		}
	default:
		s.InputBuf = append(s.InputBuf, msg.Runes...)
	}
	return nil
}
func (s *State) processKeyGo(msg tea.KeyMsg) tea.Cmd {
	s.OpBuf = s.PrevOpBuf
	switch msg.String() {
	case "g":
		s.Tree.CurrentDir.SelectFirst()
	default:
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyDelete(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y":
		err := s.Tree.DeleteMarked()
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.OpBuf = Noop
	default:
		s.OpBuf = Noop
		s.Tree.DropMark()
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyMove(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "p":
		err := s.Tree.MoveMarkedToCurrentDir()
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.OpBuf = Noop
	default:
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeyCopy(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "p":
		err := s.Tree.CopyMarkedToCurrentDir()
		if err != nil {
			s.ErrBuf = err.Error()
		}
		s.OpBuf = Noop
	default:
		return s.processKeyDefault(msg)
	}
	return nil
}
func (s *State) processKeySearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		query := string(s.InputBuf)
		s.SearchQuery = query
		s.SearchMatches = s.Tree.SearchVisible(query)
		s.SearchIdx = 0
		s.OpBuf = Noop
		s.InputBuf = []rune{}
		if len(s.SearchMatches) > 0 {
			s.Tree.NavigateToNode(s.SearchMatches[0])
		} else {
			s.ErrBuf = "no match: " + query
		}
	default:
		return s.processKeyAnyInput(msg)
	}
	return nil
}

func (s *State) navigateSearchMatch(dir int) {
	if len(s.SearchMatches) == 0 {
		return
	}
	s.SearchIdx = (s.SearchIdx + dir + len(s.SearchMatches)) % len(s.SearchMatches)
	s.Tree.NavigateToNode(s.SearchMatches[s.SearchIdx])
}

func (s *State) processKeyDefault(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		s.Tree.DropMark()
		s.OpBuf = Noop
		s.ErrBuf = ""
		s.SearchMatches = nil
		s.SearchIdx = 0
		s.SearchQuery = ""
	case "ctrl+c", "q":
		return tea.Quit
	case "shift+tab":
		s.Tree.ToggleMarkSelectedChild()
		s.Tree.SelectPreviousChild()
	case "tab":
		s.Tree.ToggleMarkSelectedChild()
		s.Tree.SelectNextChild()
	case "j", "down":
		if s.LinearNav {
			s.Tree.SelectLinearNext()
		} else {
			s.Tree.SelectNextChild()
		}
	case "k", "up":
		if s.LinearNav {
			s.Tree.SelectLinearPrev()
		} else {
			s.Tree.SelectPreviousChild()
		}
	case "pgdown":
		if s.LinearNav {
			s.Tree.SelectLinearNextN(10)
		} else {
			s.Tree.SelectNextNChildren(10)
		}
	case "pgup":
		if s.LinearNav {
			s.Tree.SelectLinearPrevN(10)
		} else {
			s.Tree.SelectPrevNChildren(10)
		}
	case "J":
		return func() tea.Msg { return PreviewScrollMsg{1} }
	case "K":
		return func() tea.Msg { return PreviewScrollMsg{-1} }
	case "shift+pgdown":
		return func() tea.Msg { return PreviewScrollMsg{20} }
	case "shift+pgup":
		return func() tea.Msg { return PreviewScrollMsg{-20} }
	case "/":
		s.SearchMatches = nil
		s.SearchIdx = 0
		s.InputBuf = []rune{}
		s.OpBuf = SearchInput
	case "n":
		s.navigateSearchMatch(1)
	case "N":
		s.navigateSearchMatch(-1)
	case "l", "right":
		err := s.Tree.SetSelectedChildAsCurrent()
		if err != nil {
			s.ErrBuf = err.Error()
		}
	case "h", "left":
		s.Tree.SetParentAsCurrent()
	case "y":
		if len(s.Tree.Marked) != 0 {
			s.OpBuf = Copy
		} else if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Copy
		}
	case "d":
		if len(s.Tree.Marked) != 0 {
			s.OpBuf = Move
		} else if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Move
		}
	case "D":
		if len(s.Tree.Marked) != 0 {
			s.OpBuf = Delete
		} else if ok := s.Tree.MarkSelectedChild(); ok {
			s.OpBuf = Delete
		}
	case "g":
		s.PrevOpBuf = s.OpBuf
		s.OpBuf = Go
	case "G":
		s.Tree.CurrentDir.SelectLast()
	case "i":
		s.Tree.DropMark()
		s.OpBuf = Insert
	case "r":
		if len(s.Tree.Marked) == 0 {
			if ok := s.Tree.MarkSelectedChild(); ok {
				s.InputBuf = []rune(s.Tree.Marked[0].Info.Name())
				s.OpBuf = Rename
			}
		}
	case "e":
		child := s.Tree.GetSelectedChild()
		if child != nil && child.Info.Mode().IsRegular() {
			return openEditor(child.Path)
		}
	case "H":
		if err := s.Tree.ToggleHiddenInCurrentDirectory(); err != nil {
			s.ErrBuf = err.Error()
		}
	case "?":
		s.HelpToggle = !s.HelpToggle
	case "enter":
		child := s.Tree.GetSelectedChild()
		if child != nil && child.Info.Mode().IsRegular() {
			return xdgOpenFile(child.Path)
		} else {
			err := s.Tree.CollapseOrExpandSelected()
			if err != nil {
				s.ErrBuf = err.Error()
			}
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		depth := int(msg.Runes[0] - '0')
		if err := s.Tree.ExpandToDepth(depth); err != nil {
			s.ErrBuf = err.Error()
		}
	case "0":
		s.Tree.CollapseAll()
	}
	return nil
}

func openEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.Command(editor, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

func xdgOpenFile(path string) tea.Cmd {
	var cmd string
	if runtime.GOOS == "darwin" {
		cmd = "open"
	} else {
		cmd = "xdg-open"
	}
	c := exec.Command(cmd, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}
