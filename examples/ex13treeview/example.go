package ex13treeview

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows a TreeView with columns: a project tree whose
// folders load their contents when first opened, editable names and
// comments, multiselect and a context menu.
//
// Try: arrows (Left/Right collapse/expand), Home/End, typing a name,
// F2 or a letter-key to rename, Enter/double click to open, Ctrl+click and
// Shift+click with multiselect, Ctrl+C, the right mouse button.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("TreeView")
	form.SetSize(760, 520)
	panel := form.Panel()

	status := panel.AddLabel(0, 0, "F2 or the right-click menu edits a name or a comment; double click opens")

	tree := ui.NewTreeView()
	tree.SetColumnCount(3)
	tree.SetColumnName(0, "Name")
	tree.SetColumnName(1, "Size")
	tree.SetColumnName(2, "Comment")
	tree.SetColumnWidth(0, 280)
	tree.SetColumnWidth(1, 90)
	tree.SetColumnReadOnly(1, true)
	tree.SetEditTriggerF2(true)
	tree.SetEditTriggerKeyDown(false)
	panel.AddWidget(1, 0, tree)

	folderIcon := icon(color.RGBA{0xF5, 0xB7, 0x31, 0xFF})
	fileIcon := icon(color.RGBA{0x64, 0x9B, 0xD8, 0xFF})

	addFolder := func(parent *ui.TreeNode, name string) *ui.TreeNode {
		n := tree.AddNode(parent, name)
		n.SetImage(folderIcon)
		n.SetHasChildren(true) // loaded in SetOnExpand
		n.SetReadOnly(1, true)
		return n
	}
	addFile := func(parent *ui.TreeNode, name string, size int) *ui.TreeNode {
		n := tree.AddNode(parent, name)
		n.SetImage(fileIcon)
		n.SetText(1, fmt.Sprintf("%d KB", size))
		return n
	}

	// Folders get their contents when opened the first time
	loaded := map[*ui.TreeNode]bool{}
	tree.SetOnExpand(func(node *ui.TreeNode) {
		if loaded[node] {
			return
		}
		loaded[node] = true
		for i := 1; i <= 3; i++ {
			addFolder(node, fmt.Sprintf("%s-sub%d", node.Text(0), i))
		}
		for i := 1; i <= 5; i++ {
			addFile(node, fmt.Sprintf("file%d.go", i), i*7)
		}
		node.SetHasChildren(node.ChildCount() > 0)
	})

	src := addFolder(nil, "src")
	addFolder(nil, "docs")
	addFolder(nil, "assets")
	addFile(nil, "README.md", 4).SetText(2, "Start here")
	addFile(nil, "go.mod", 1)
	src.Expand()

	tree.SetOnSelectionChanged(func(node *ui.TreeNode) {
		names := []string{}
		for _, n := range tree.SelectedNodes() {
			names = append(names, n.Text(0))
		}
		status.SetText(fmt.Sprintf("Selected (%d): %s", len(names), strings.Join(names, ", ")))
	})
	tree.SetOnNodeActivated(func(node *ui.TreeNode) {
		if !node.HasChildren() {
			status.SetText("Open " + node.Text(0))
		}
	})
	tree.SetOnCellChanged(func(node *ui.TreeNode, col int, text string) bool {
		if col == 0 && strings.TrimSpace(text) == "" {
			status.SetText("A name can't be empty")
			return false
		}
		status.SetText(fmt.Sprintf("%s: column %d = %q", node.Text(0), col, text))
		return true
	})

	menu := ui.NewContextMenu(tree)
	menu.AddItem("Rename", func() {
		tree.SetCurrentColumn(0)
		tree.EditCurrentCell("")
	})
	menu.AddItem("Edit Comment", func() {
		tree.SetCurrentColumn(2)
		tree.EditCurrentCell("")
	})
	menu.AddSeparator()
	menu.AddItem("New File", func() {
		parent := tree.CurrentNode()
		if parent != nil && !parent.HasChildren() {
			parent = parent.Parent()
		}
		if parent != nil {
			parent.Expand()
		}
		n := addFile(parent, "new.txt", 0)
		tree.SetCurrentNode(n)
		tree.SetCurrentColumn(0)
		tree.EditCurrentCell("")
	})
	menu.AddItem("Delete", func() {
		for _, n := range tree.SelectedNodes() {
			n.Remove()
		}
	})
	menu.AddSeparator()
	menu.AddItem("Copy", tree.CopySelectionToClipboard)
	tree.SetContextMenu(menu)

	buttons := panel.AddPanel(2, 0)
	buttons.AddButton(0, 0, "Expand All", tree.ExpandAll)
	buttons.AddButton(0, 1, "Collapse All", tree.CollapseAll)
	multi := ui.NewCheckbox("Multiselect")
	multi.SetOnStateChanged(func() { tree.SetMultiselect(multi.Checked()) })
	buttons.AddWidget(0, 2, multi)
	grid := ui.NewCheckbox("Grid lines")
	grid.SetOnStateChanged(func() { tree.SetGridLines(grid.Checked()) })
	buttons.AddWidget(0, 3, grid)
	typing := ui.NewCheckbox("Type to rename")
	typing.SetOnStateChanged(func() { tree.SetEditTriggerKeyDown(typing.Checked()) })
	buttons.AddWidget(0, 4, typing)
	buttons.AddHSpacer(0, 5)

	return form
}

// icon draws a small rounded square.
func icon(col color.RGBA) image.Image {
	const size = 16
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 2; y < size-2; y++ {
		for x := 1; x < size-1; x++ {
			if (x == 1 || x == size-2) && (y == 2 || y == size-3) {
				continue
			}
			img.Set(x, y, col)
		}
	}
	return img
}
