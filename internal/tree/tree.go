package tree

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type Tree struct {
	Root       *Node
	CurrentDir *Node
	Marked     []*Node

	sortingFunc NodeSortingFunc
	watcher     *fsnotify.Watcher
}

func (t *Tree) GetSelectedChild() *Node {
	if len(t.CurrentDir.Children) > 0 {
		return t.CurrentDir.Children[t.CurrentDir.selectedChildIdx]
	}
	return nil
}
func (t *Tree) ToggleHiddenInCurrentDirectory() error {
	t.CurrentDir.showHidden = !t.CurrentDir.showHidden
	return t.CurrentDir.readChildren(defaultNodeSorting)
}
func (t *Tree) RemoveNodeFromMarkByPath(path string) {
	t.Marked = slices.DeleteFunc(
		t.Marked,
		func(n *Node) bool { return n.Path == path },
	)
}
func (t *Tree) RefreshNodeParentByPath(path string) error {
	parentDir := filepath.Dir(path)
	cur := t.Root
outer:
	for {
		// Reading children when a parent node found.
		if parentDir == cur.Path {
			return cur.readChildren(t.sortingFunc)
		}
		// Going through directories towards `parentDir`.
		for _, ch := range cur.Children {
			if strings.HasPrefix(path, ch.Path) {
				cur = ch
				continue outer
			}
		}
		return nil
	}
}
func (t *Tree) RenameMarked(newName string) error {
	if len(t.Marked) != 1 {
		return nil
	}
	marked := t.Marked[0]

	if newName == "" {
		return fmt.Errorf("new name must not be empty")
	}
	targetPath := filepath.Join(marked.Parent.Path, newName)
	// NOTE: probably not the best way to check if file exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("file %s already exists", newName)
	} else if !os.IsNotExist(err) {
		return err
	}
	err := os.Rename(marked.Path, targetPath)
	if err != nil {
		return err
	}
	t.Marked = nil
	return nil
}
func (t *Tree) CreateFileInCurrent(name string) error {
	_, err := os.Create(filepath.Join(t.CurrentDir.Path, name))
	return err
}
func (t *Tree) CreateDirectoryInCurrent(name string) error {
	return os.Mkdir(filepath.Join(t.CurrentDir.Path, name), os.ModePerm)
}
func (t *Tree) SelectNextChild() {
	if t.CurrentDir.selectedChildIdx < len(t.CurrentDir.Children)-1 {
		t.CurrentDir.selectedChildIdx += 1
	}
}
func (t *Tree) SelectPreviousChild() {
	if t.CurrentDir.selectedChildIdx > 0 {
		t.CurrentDir.selectedChildIdx -= 1
	}
}
func (t *Tree) SelectNextNChildren(n int) {
	max := len(t.CurrentDir.Children) - 1
	t.CurrentDir.selectedChildIdx = min(t.CurrentDir.selectedChildIdx+n, max)
}
func (t *Tree) SelectPrevNChildren(n int) {
	t.CurrentDir.selectedChildIdx = max(t.CurrentDir.selectedChildIdx-n, 0)
}

// VisibleNodes returns every node in DFS render order that is currently
// visible (parent's Children != nil), excluding Root itself.
func (t *Tree) VisibleNodes() []*Node {
	var nodes []*Node
	var walk func(*Node)
	walk = func(node *Node) {
		for _, child := range node.Children {
			nodes = append(nodes, child)
			if child.Children != nil {
				walk(child)
			}
		}
	}
	walk(t.Root)
	return nodes
}

// SelectLinearNext moves selection one step forward across all visible rows.
func (t *Tree) SelectLinearNext() {
	t.selectLinearN(1)
}

// SelectLinearPrev moves selection one step backward across all visible rows.
func (t *Tree) SelectLinearPrev() {
	t.selectLinearN(-1)
}

// SelectLinearNextN moves selection n steps forward across all visible rows.
func (t *Tree) SelectLinearNextN(n int) {
	t.selectLinearN(n)
}

// SelectLinearPrevN moves selection n steps backward across all visible rows.
func (t *Tree) SelectLinearPrevN(n int) {
	t.selectLinearN(-n)
}

func (t *Tree) selectLinearN(delta int) {
	nodes := t.VisibleNodes()
	selected := t.GetSelectedChild()
	for i, n := range nodes {
		if n == selected {
			target := max(0, min(i+delta, len(nodes)-1))
			t.NavigateToNode(nodes[target])
			return
		}
	}
}

// SearchVisible returns all currently expanded nodes whose names contain query
// (case-insensitive), walking the full tree from Root.
func (t *Tree) SearchVisible(query string) []*Node {
	if query == "" {
		return nil
	}
	query = strings.ToLower(query)
	var matches []*Node
	var walk func(*Node)
	walk = func(node *Node) {
		for _, child := range node.Children {
			if strings.Contains(strings.ToLower(child.Info.Name()), query) {
				matches = append(matches, child)
			}
			if child.Children != nil {
				walk(child)
			}
		}
	}
	walk(t.Root)
	return matches
}

// NavigateToNode sets CurrentDir to node's parent and selects node.
func (t *Tree) NavigateToNode(node *Node) {
	if node == nil || node.Parent == nil {
		return
	}
	t.CurrentDir = node.Parent
	for i, ch := range node.Parent.Children {
		if ch == node {
			node.Parent.selectedChildIdx = i
			break
		}
	}
}
func (t *Tree) SetSelectedChildAsCurrent() error {
	selectedChild := t.GetSelectedChild()
	if selectedChild == nil {
		return nil
	}
	if !selectedChild.Info.IsDir() {
		return nil
	}
	if selectedChild.Children == nil {
		err := selectedChild.readChildren(t.sortingFunc)
		if err != nil {
			return err
		}
		t.watcher.Add(selectedChild.Path)
	}
	t.CurrentDir = selectedChild
	return nil
}
func (t *Tree) SetParentAsCurrent() {
	if t.CurrentDir.Parent != nil {
		currentName := t.CurrentDir.Info.Name()
		// setting parent current to match the directory, that we're leaving
		newParentIdx := slices.IndexFunc(t.CurrentDir.Parent.Children, func(n *Node) bool { return n.Info.Name() == currentName })
		t.CurrentDir.Parent.selectedChildIdx = newParentIdx

		t.CurrentDir = t.CurrentDir.Parent
	}
}
func (t *Tree) ToggleMarkSelectedChild() bool {
	if selected := t.GetSelectedChild(); selected != nil {
		if !slices.Contains(t.Marked, selected) {
			t.Marked = append(t.Marked, selected)
		} else {
			t.Marked = slices.DeleteFunc(t.Marked, func(n *Node) bool { return n == selected })
		}
		return true
	}
	return false
}
func (t *Tree) MarkSelectedChild() bool {
	if selected := t.GetSelectedChild(); selected != nil {
		if !slices.Contains(t.Marked, selected) {
			t.Marked = append(t.Marked, selected)
		}
		return true
	}
	return false
}
func (t *Tree) DropMark() {
	t.Marked = nil
}
func (t *Tree) DeleteMarked() error {
	if t.Marked == nil {
		return nil
	}
	for _, marked := range t.Marked {
		cmd := exec.Command("rm", "-r", marked.Path)
		err := cmd.Run()
		if err != nil {
			return err // todo: this is not the same error...?
		}
	}
	t.Marked = nil
	return nil
}
func (t *Tree) CopyMarkedToCurrentDir() error {
	if t.Marked == nil {
		return nil
	}
	targetDir := t.CurrentDir.Path
	for _, marked := range t.Marked {
		targetFileName, err := generateNewFileName(marked.Info.Name(), targetDir)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, targetFileName)

		cmd := exec.Command("cp", "-r", marked.Path, targetPath)
		err = cmd.Run()
		if err != nil {
			return err // todo: this is not the same error...?
		}
	}
	t.Marked = nil
	return nil
}
func (t *Tree) MoveMarkedToCurrentDir() error {
	if t.Marked == nil {
		return nil
	}
	targetDir := t.CurrentDir.Path
	for _, marked := range t.Marked {
		targetFileName, err := generateNewFileName(marked.Info.Name(), targetDir)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, targetFileName)

		cmd := exec.Command("mv", "-n", marked.Path, targetPath)
		err = cmd.Run()
		if err != nil {
			return err // todo: this is not the same error...?
		}
	}
	t.Marked = nil
	return nil
}
func (t *Tree) CollapseOrExpandSelected() error {
	selectedChild := t.GetSelectedChild()
	if selectedChild == nil {
		return nil
	}
	if selectedChild.Children != nil {
		selectedChild.orphanChildren()
		t.watcher.Remove(selectedChild.Path)
	} else {
		err := selectedChild.readChildren(t.sortingFunc)
		if err != nil {
			return err
		}
		t.watcher.Add(selectedChild.Path)
	}
	return nil
}

// collapseRecursive recursively collapses node and all its expanded descendants.
func (t *Tree) collapseRecursive(node *Node) {
	if node == nil || node.Children == nil {
		return
	}
	for _, child := range node.Children {
		if child.Info.IsDir() {
			t.collapseRecursive(child)
		}
	}
	node.orphanChildren()
	t.watcher.Remove(node.Path)
}

// CollapseAll collapses all expanded directories within CurrentDir.
func (t *Tree) CollapseAll() {
	for _, child := range t.CurrentDir.Children {
		t.collapseRecursive(child)
	}
}

// ExpandToDepth expands directories within CurrentDir to the given depth.
// depth=1 shows only immediate children (all sub-dirs collapsed),
// depth=2 expands one level of sub-dirs, etc. Mirrors tree -L N behaviour.
func (t *Tree) ExpandToDepth(depth int) error {
	return t.expandNodeToDepth(t.CurrentDir, depth)
}

func (t *Tree) expandNodeToDepth(node *Node, depth int) error {
	if node == nil || !node.Info.IsDir() {
		return nil
	}
	if node.Children == nil {
		if err := node.readChildren(t.sortingFunc); err != nil {
			return err
		}
		t.watcher.Add(node.Path)
	}
	for _, child := range node.Children {
		if !child.Info.IsDir() {
			continue
		}
		if depth <= 1 {
			t.collapseRecursive(child)
		} else {
			if err := t.expandNodeToDepth(child, depth-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func InitTree(dir string, sortingFunc NodeSortingFunc) (*Tree, <-chan NodeChange, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, err
	}

	rootInfo, err := os.Lstat(absDir)
	if err != nil {
		return nil, nil, err
	}
	if !rootInfo.IsDir() {
		return nil, nil, fmt.Errorf("%s is not a directory", absDir)
	}
	if sortingFunc == nil {
		sortingFunc = defaultNodeSorting
	}

	root := NewNode(absDir, rootInfo, nil)

	err = root.readChildren(sortingFunc)
	if err != nil {
		return nil, nil, err
	}
	if len(root.Children) == 0 {
		return nil, nil, fmt.Errorf("Can't initialize on empty directory '%s'", absDir)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}
	changeChan := runFSWatcher(watcher)
	err = watcher.Add(root.Path)
	if err != nil {
		return nil, nil, err
	}

	tree := &Tree{
		Root:        root,
		CurrentDir:  root,
		sortingFunc: sortingFunc,
		watcher:     watcher,
	}
	return tree, changeChan, nil
}

// Checks if fname already exists in targetDir.
// Adds "copy_" prefix (multiple times), until new file name becomes unique in derecotry.
func generateNewFileName(fname, targetDir string) (string, error) {
	currentDirContent, err := os.ReadDir(targetDir)
	if err != nil {
		return "", err
	}
	for slices.ContainsFunc(currentDirContent, func(e fs.DirEntry) bool { return e.Name() == fname }) {
		fname = "copy_" + fname
	}
	return fname, nil
}
