package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// The folder tree: the home folder and the root, each folder's subfolders
// are read when it is expanded the first time. A node's data is its path.

func (e *explorer) newTree() *ui.TreeView {
	t := ui.NewTreeView()
	t.SetColumnCount(1)
	t.SetHeaderVisible(false)
	t.SetEditTriggerDoubleClick(false)
	t.SetEditTriggerEnter(false)
	t.SetEditTriggerF2(false)
	t.SetEditTriggerKeyDown(false)
	t.SetMinWidth(120)

	t.SetOnExpand(e.loadFolders)
	t.SetOnSelectionChanged(func(node *ui.TreeNode) {
		if e.syncTree || node == nil {
			return
		}
		if path, ok := node.Data().(string); ok && path != e.dir {
			e.navigate(path, true)
		}
	})
	e.tree = t
	e.reloadTree()
	return t
}

// reloadTree builds the tree again, e.g. when hidden folders are shown or
// hidden, keeping the current folder selected.
func (e *explorer) reloadTree() {
	e.tree.Clear()
	if home, err := os.UserHomeDir(); err == nil {
		e.addFolder(nil, "Home", home).Expand()
	}
	e.addFolder(nil, "/", "/")
	if e.dir != "" {
		e.selectInTree(e.dir)
	}
}

func (e *explorer) addFolder(parent *ui.TreeNode, name, path string) *ui.TreeNode {
	n := e.tree.AddNode(parent, name)
	n.SetImage(icons.Folder(icons.Size))
	n.SetData(path)
	n.SetHasChildren(true) // until loadFolders finds out
	return n
}

// loadFolders adds the subfolders of a node, the first time it is expanded.
func (e *explorer) loadFolders(node *ui.TreeNode) {
	if node.ChildCount() > 0 {
		return
	}
	dir := node.Data().(string)
	list, err := os.ReadDir(dir)
	if err != nil {
		node.SetHasChildren(false)
		e.showError(err)
		return
	}
	var names []string
	for _, d := range list {
		if !e.showHidden && isHidden(d.Name()) {
			continue
		}
		path := filepath.Join(dir, d.Name())
		if d.IsDir() || d.Type()&os.ModeSymlink != 0 && isDir(path) {
			names = append(names, d.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
	for _, name := range names {
		e.addFolder(node, name, filepath.Join(dir, name))
	}
	node.SetHasChildren(len(names) > 0)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// selectInTree selects the node of the folder dir, opening the folders on
// the way; under the home folder the Home branch is used. If a folder on
// the way isn't in the tree (hidden), its nearest parent is selected.
func (e *explorer) selectInTree(dir string) {
	var best *ui.TreeNode
	for _, root := range e.tree.Nodes() {
		path := root.Data().(string)
		if (dir == path || strings.HasPrefix(dir, strings.TrimSuffix(path, "/")+"/")) &&
			(best == nil || len(path) > len(best.Data().(string))) {
			best = root
		}
	}
	if best == nil {
		return
	}
	rel, err := filepath.Rel(best.Data().(string), dir)
	if err != nil {
		return
	}
	node := best
	if rel != "." {
	parts:
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			node.Expand() // loads the subfolders
			for _, child := range node.Children() {
				if child.Text(0) == part {
					node = child
					continue parts
				}
			}
			break
		}
	}
	e.syncTree = true
	e.tree.SetCurrentNode(node)
	e.tree.ScrollToNode(node)
	e.syncTree = false
}
