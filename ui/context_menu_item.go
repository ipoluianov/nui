package ui

import (
	"image"
	"image/color"
	"time"

	"github.com/nfnt/resize"
)

type ContextMenuItem struct {
	Widget
	text                 string
	image                image.Image
	OnClick              func()
	parentMenu           *ContextMenu
	needToClosePopupMenu func()

	timerEnabled bool

	timerLastElapsedDTMSec int64

	innerMenu *ContextMenu

	// separator: a line between groups of items, not an item to click
	separator bool

	// pressed is set by a press on the item; the release over it clicks it
	pressed bool

	// shortcut runs the item from the keyboard and is shown on its right,
	// see SetShortcut. mnemonic is the underlined letter ("&Open"), at
	// mnemonicIndex of text.
	shortcut      Shortcut
	mnemonic      rune
	mnemonicIndex int
}

// ContextMenuSeparatorHeight is the height of a separator line between groups of items
const ContextMenuSeparatorHeight = 9

// IsSeparator reports whether the item is a separator line
func (c *ContextMenuItem) IsSeparator() bool {
	return c.separator
}

// height is the height the item takes in the menu
func (c *ContextMenuItem) height() int {
	if c.separator {
		return ContextMenuSeparatorHeight
	}
	return ThemeRowHeight()
}

func NewContextMenuItem() *ContextMenuItem {
	var item ContextMenuItem
	item.InitWidget()
	item.SetAbsolutePositioning(true)
	item.SetMouseCursor(MouseCursorPointer)

	item.SetOnPaint(item.Draw)

	item.SetOnMouseDown(item.mouseDownHandler)
	item.SetOnMouseUp(item.mouseUpHandler)
	item.SetOnMouseMove(item.MouseMove)
	item.SetOnMouseEnter(item.MouseEnter)
	item.SetOnMouseLeave(item.MouseLeave)

	item.AddTimer(200, item.timerShowInnerMenuHandler)
	return &item
}

// SetText sets the text; "&" marks the mnemonic letter: "&Open" shows
// "Open" with O underlined, and O chooses the item in the open menu.
func (c *ContextMenuItem) SetText(text string) {
	c.text, c.mnemonic, c.mnemonicIndex = parseMnemonic(text)
	c.form.Update()
}

// Text returns the text as shown, without the mnemonic mark
func (c *ContextMenuItem) Text() string {
	return c.text
}

// SetShortcut sets the keys that run the item without opening the menu,
// shown on the item's right: "Ctrl+S", "Ctrl+Shift+Z", "F5", "Mod+S" (Cmd
// on macOS, Ctrl elsewhere). Works for the items of the form's main menu.
// Panics on a wrong shortcut (see ParseShortcut); "" removes it. Returns the
// item for chaining:
//
//	file.AddItem("&Save", onSave).SetShortcut("Mod+S")
func (c *ContextMenuItem) SetShortcut(shortcut string) *ContextMenuItem {
	c.shortcut = Shortcut{}
	if shortcut != "" {
		c.shortcut = MustParseShortcut(shortcut)
	}
	c.form.Update()
	return c
}

// Shortcut returns the item's shortcut, zero if none
func (c *ContextMenuItem) Shortcut() Shortcut {
	return c.shortcut
}

// selectable reports whether the keyboard can choose the item
func (c *ContextMenuItem) selectable() bool {
	return !c.separator && c.IsVisible() && c.Enabled()
}

// activateByKeyboard opens the item's submenu, with its first item chosen,
// or clicks the item
func (c *ContextMenuItem) activateByKeyboard() {
	if c.innerMenu != nil {
		x, y := c.RectClientAreaOnWindow()
		c.innerMenu.showMenu(x+c.Width(), y, c.parentMenu)
		c.innerMenu.activateFirst()
		return
	}
	c.click()
}

// isActive reports whether the item is highlighted: under the mouse, or
// chosen with the keyboard
func (c *ContextMenuItem) isActive() bool {
	return c.IsHovered() || (c.parentMenu != nil && c.parentMenu.active == c)
}

// SetImage sets the icon shown left of the text; a larger image is scaled down
// to ContextMenuItemIconSize. nil removes the icon. Returns the item for chaining:
//
//	menu.AddItem("Edit", onEdit).SetImage(editIcon)
func (c *ContextMenuItem) SetImage(img image.Image) *ContextMenuItem {
	if img != nil {
		b := img.Bounds()
		if b.Dx() > ContextMenuItemIconSize || b.Dy() > ContextMenuItemIconSize {
			img = resize.Thumbnail(ContextMenuItemIconSize, ContextMenuItemIconSize, img, resize.Lanczos3)
		}
	}
	c.image = img
	c.form.Update()
	return c
}

func (c *ContextMenuItem) Image() image.Image {
	return c.image
}

func (c *ContextMenuItem) ControlType() string {
	return "PopupMenuItem"
}

// contextMenuItemPadding keeps the item's text and submenu arrow off its
// edges - the item itself still spans the full menu width so the hover
// highlight reaches border to border, as in standard menu styling.
const contextMenuItemPadding = 10

// ContextMenuItemIconSize is the size of the item icons
const ContextMenuItemIconSize = 16

// textX returns where the text starts: after the icon column when any item of the menu has an icon,
// so the texts of all the items stay aligned
func (c *ContextMenuItem) textX() int {
	if c.parentMenu != nil && c.parentMenu.hasImages() {
		return contextMenuItemPadding*2 + ContextMenuItemIconSize
	}
	return contextMenuItemPadding
}

// Draw: the hovered item in the accent color, like in the combo box dropdown
func (c *ContextMenuItem) Draw(ctx *Canvas) {
	p := CurrentPalette()
	if c.separator {
		ctx.FillRect(0, 0, c.InnerWidth(), c.InnerHeight(), p.PopupBase)
		ctx.FillRect(contextMenuItemPadding, c.Height()/2, c.Width()-contextMenuItemPadding*2, 1, p.Divider)
		return
	}

	backColor, textColor := p.PopupBase, p.Text
	if c.isActive() && c.Enabled() {
		backColor, textColor = p.Highlight, p.HighlightedText
	}
	if !c.Enabled() {
		textColor = p.DisabledText
	}
	ctx.FillRect(0, 0, c.InnerWidth(), c.InnerHeight(), backColor)

	if c.image != nil {
		b := c.image.Bounds()
		ctx.DrawImage(contextMenuItemPadding+(ContextMenuItemIconSize-b.Dx())/2, (c.Height()-b.Dy())/2, c.image)
	}

	textX := c.textX()
	textAreaWidth := c.Width() - textX - contextMenuItemPadding
	if c.innerMenu != nil {
		textAreaWidth -= c.Height() + contextMenuItemPadding
	}
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	ctx.SetVAlign(VAlignCenter)

	// The shortcut on the right, softer than the text
	if shortcut := c.shortcut.String(); shortcut != "" && c.innerMenu == nil {
		shortcutWidth, _, _ := MeasureText(c.FontFamily(), c.FontSize(), shortcut)
		ctx.SetHAlign(HAlignRight)
		ctx.SetColor(MixColors(colorToRGBA(textColor), backColor, 0.35))
		ctx.DrawText(textX, 0, textAreaWidth, c.Height(), shortcut)
		textAreaWidth -= shortcutWidth + contextMenuShortcutGap
	}

	displayText := truncateTextToWidth(c.FontFamily(), c.FontSize(), c.text, textAreaWidth)
	ctx.SetHAlign(HAlignLeft)
	ctx.SetColor(textColor)
	ctx.DrawText(textX, 0, textAreaWidth, c.Height(), displayText)
	if displayText == c.text {
		drawMnemonicUnderline(ctx, textX, 0, textAreaWidth, c.Height(), c.text, c.mnemonicIndex)
	}

	if c.innerMenu != nil {
		c.drawSubmenuArrow(ctx, textColor)
	}
}

// contextMenuArrowHalfHeight and contextMenuArrowWidth size the submenu arrow
const contextMenuArrowHalfHeight = 4
const contextMenuArrowWidth = 5

// drawSubmenuArrow draws a small right-pointing triangle at the right edge
// of an item that opens a submenu.
func (c *ContextMenuItem) drawSubmenuArrow(ctx *Canvas, arrowColor color.Color) {
	right := c.Width() - contextMenuItemPadding - 2
	left := right - contextMenuArrowWidth
	midY := c.Height() / 2
	ctx.FillTriangle(left, midY-contextMenuArrowHalfHeight, left, midY+contextMenuArrowHalfHeight, right, midY, arrowColor)
}

func (c *ContextMenuItem) mouseDownHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	c.timerEnabled = false
	if c.separator {
		return true // a click on a separator does nothing and keeps the menu open
	}

	// A submenu opens at once; a command runs on the release (mouseUpHandler),
	// like in the system menus, so a press can still be taken back by moving
	// off the item
	if c.innerMenu != nil {
		x, y := c.RectClientAreaOnWindow()
		w := c.Width()
		c.innerMenu.showMenu(x+w, y, c.parentMenu)
		return true
	}
	c.pressed = button == MouseButtonLeft
	return true
}

// mouseUpHandler: the release after a press on the item clicks the item
// under the mouse - this one, or another one the mouse was moved to with the
// button down, also in a submenu. Released off the menus: no click.
func (c *ContextMenuItem) mouseUpHandler(button MouseButton, x int, y int, mods KeyModifiers) bool {
	pressed := c.pressed
	c.pressed = false
	if !pressed || button != MouseButtonLeft || c.form == nil {
		return true
	}
	wx, wy := c.RectClientAreaOnWindow()
	if item := c.form.contextMenuItemAt(wx+x, wy+y); item != nil {
		item.click()
	}
	return true
}

// contextMenuItemAt returns the clickable menu item at the client point in
// the open menus, nil if there is none
func (c *Form) contextMenuItemAt(x, y int) *ContextMenuItem {
	h := c.popupHostAt(x, y)
	if h == nil {
		return nil
	}
	if _, ok := h.widget.(*ContextMenu); !ok {
		return nil
	}
	item, ok := h.widget.findWidgetAt(x-h.widget.X(), y-h.widget.Y()).(*ContextMenuItem)
	if !ok || item.separator || item.innerMenu != nil || !item.Enabled() {
		return nil
	}
	return item
}

// click closes the menu and runs the item's command
func (c *ContextMenuItem) click() {
	if c.needToClosePopupMenu != nil {
		c.needToClosePopupMenu()
	}

	if c.OnClick != nil {
		c.OnClick()
	}
}

func (c *ContextMenuItem) MouseEnter() {
	// The keyboard goes on from the item under the mouse
	if c.parentMenu != nil && c.selectable() {
		c.parentMenu.active = c
	}
	c.form.Panel().CloseAfterPopupWidget(c.parentMenu)
	if c.innerMenu != nil {
		c.timerEnabled = true
		c.timerLastElapsedDTMSec = time.Now().UnixNano() / 1000000
		return
	}
}

func (c *ContextMenuItem) MouseLeave() {
	c.timerEnabled = false
}

func (c *ContextMenuItem) MouseMove(x int, y int, mods KeyModifiers) bool {
	c.form.Update()
	return true
}

func (c *ContextMenuItem) SetInnerMenu(menu *ContextMenu) {
	c.innerMenu = menu
}

// attachToForm also attaches innerMenu, which AddItemWithSubmenu stores
// directly on the item rather than adding it as a child widget, so the
// generic Widget.attachToForm cascade would otherwise never reach it -
// leaving its form nil and crashing (nil c.form.Panel()) the first time
// the submenu is opened.
func (c *ContextMenuItem) attachToForm(self Widgeter, form *Form) {
	c.Widget.attachToForm(self, form)
	if c.innerMenu != nil {
		c.innerMenu.attachToForm(c.innerMenu, form)
	}
}

func (c *ContextMenuItem) timerShowInnerMenuHandler() {
	if c.timerEnabled && time.Now().UnixNano()/1000000-c.timerLastElapsedDTMSec > 200 {
		c.timerEnabled = false
		x, y := c.parentMenu.RectClientAreaOnWindow()
		y += c.Y()
		w := c.Width()
		c.innerMenu.showMenu(x+w, y, c.parentMenu)
	}
}
