package ui

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// The layout of a widget in XML (SetLayout): an element a widget, its
// attributes the widget's properties, its children the widgets in it.
//
//	<column padding="8">
//		<row>
//			<label text="Name:"/>
//			<textbox id="name" hint="Your name"/>
//		</row>
//		<checkbox id="remember" text="Remember me" checked="true"/>
//		<vspacer/>
//		<row><hspacer/><button id="ok" text="OK" onclick="OK"/></row>
//	</column>
//
// A <column> puts its children one under another, a <row> one after
// another. The elements and the attributes of each are listed in
// doc/ui/content/xml_layout.md; a mistake (an unknown element or
// attribute, a value that is not a number, a handler not found) doesn't
// stop the layout: the rest is built, and SetLayout returns all the
// mistakes in a *LayoutError, with their lines.

type uiNode struct {
	XMLName   xml.Name
	Attrs     []xml.Attr
	Nodes     []uiNode
	InnerText string
	// Line is the line of the element in the XML
	Line int
}

func (c *uiNode) GetChildByName(name string) *uiNode {
	for index, child := range c.Nodes {
		if child.XMLName.Local == name {
			return &c.Nodes[index]
		}
	}
	return nil
}

func (c *uiNode) GetAttrByName(name string) *xml.Attr {
	for index, attr := range c.Attrs {
		if attr.Name.Local == name {
			return &c.Attrs[index]
		}
	}
	return nil
}

func (c *uiNode) GetAttrValueByName(name string, defaultValue string) string {
	for _, attr := range c.Attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return defaultValue
}

func (c *uiNode) GetAttrValueByNameInt(name string, defaultValue int) int {
	for _, attr := range c.Attrs {
		if attr.Name.Local == name {
			if intValue, err := strconv.Atoi(attr.Value); err == nil {
				return intValue
			}
			return defaultValue
		}
	}
	return defaultValue
}

// parseLayoutXML reads the XML into nodes, keeping the line of each element
func parseLayoutXML(text string) (*uiNode, error) {
	d := xml.NewDecoder(strings.NewReader(text))
	var stack []*uiNode
	var root *uiNode
	for {
		line, _ := d.InputPos()
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &uiNode{XMLName: t.Name, Attrs: slices.Clone(t.Attr), Line: line}
			if len(stack) == 0 {
				if root != nil {
					return nil, fmt.Errorf("line %d: a second root element <%s>", line, t.Name.Local)
				}
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Nodes = append(parent.Nodes, *n)
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].InnerText += string(t)
			}
		}
	}
	if root == nil {
		return nil, errors.New("no element")
	}
	return root, nil
}

// LayoutError is the mistakes found in the XML of a layout
type LayoutError struct {
	Problems []LayoutProblem
}

// LayoutProblem is a mistake at a line of the XML of a layout
type LayoutProblem struct {
	Line    int
	Element string
	Message string
}

func (e *LayoutError) Error() string {
	var lines []string
	for _, p := range e.Problems {
		lines = append(lines, fmt.Sprintf("line %d: <%s>: %s", p.Line, p.Element, p.Message))
	}
	return "layout: " + strings.Join(lines, "; ")
}

// layoutBuilder builds the widgets of the nodes
type layoutBuilder struct {
	handlers any
	widgets  map[string]Widgeter
	ids      map[string]int // the line of each id, to find the same one twice
	problems []LayoutProblem
}

func (b *layoutBuilder) problem(n *uiNode, format string, args ...any) {
	b.problems = append(b.problems, LayoutProblem{Line: n.Line, Element: n.XMLName.Local, Message: fmt.Sprintf(format, args...)})
}

// xmlAttr sets an attribute on the widget of an element
type xmlAttr func(w Widgeter, value string) error

// xmlElement is what an element of the XML makes
type xmlElement struct {
	make func() Widgeter
	// attrs are the attributes of the element, besides the common ones
	attrs map[string]xmlAttr
	// events are the handler attributes the widget calls: onclick...
	events []string
	// children: how the child elements are placed - "column" (one under
	// another), "row", "none" (no children), "tabs", "table", "split" (two)
	children string
}

func parseBool(s string) (bool, error) {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("%q is not true or false", s)
	}
	return v, nil
}

func parseInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%q is not a whole number", s)
	}
	return v, nil
}

func parseFloat(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	return v, nil
}

func boolAttr(set func(w Widgeter, v bool)) xmlAttr {
	return func(w Widgeter, s string) error {
		v, err := parseBool(s)
		if err == nil {
			set(w, v)
		}
		return err
	}
}

func intAttr(set func(w Widgeter, v int)) xmlAttr {
	return func(w Widgeter, s string) error {
		v, err := parseInt(s)
		if err == nil {
			set(w, v)
		}
		return err
	}
}

func floatAttr(set func(w Widgeter, v float64)) xmlAttr {
	return func(w Widgeter, s string) error {
		v, err := parseFloat(s)
		if err == nil {
			set(w, v)
		}
		return err
	}
}

func textAttr(set func(w Widgeter, v string)) xmlAttr {
	return func(w Widgeter, s string) error {
		set(w, s)
		return nil
	}
}

// commonXMLAttrs are the attributes of every element
var commonXMLAttrs = map[string]xmlAttr{
	"enabled":        boolAttr(func(w Widgeter, v bool) { w.SetEnabled(v) }),
	"visible":        boolAttr(func(w Widgeter, v bool) { w.widgetBase().SetVisible(v) }),
	"tooltip":        textAttr(func(w Widgeter, v string) { w.widgetBase().SetTooltip(v) }),
	"minwidth":       intAttr(func(w Widgeter, v int) { w.widgetBase().SetMinWidth(v) }),
	"minheight":      intAttr(func(w Widgeter, v int) { w.widgetBase().SetMinHeight(v) }),
	"maxwidth":       intAttr(func(w Widgeter, v int) { w.widgetBase().SetMaxWidth(v) }),
	"maxheight":      intAttr(func(w Widgeter, v int) { w.widgetBase().SetMaxHeight(v) }),
	"width":          intAttr(func(w Widgeter, v int) { w.widgetBase().SetMinWidth(v); w.widgetBase().SetMaxWidth(v) }),
	"height":         intAttr(func(w Widgeter, v int) { w.widgetBase().SetMinHeight(v); w.widgetBase().SetMaxHeight(v) }),
	"xexpandable":    boolAttr(func(w Widgeter, v bool) { w.widgetBase().SetXExpandable(v) }),
	"yexpandable":    boolAttr(func(w Widgeter, v bool) { w.widgetBase().SetYExpandable(v) }),
	"fontsize":       floatAttr(func(w Widgeter, v float64) { w.widgetBase().SetFontSize(v) }),
	"fontbold":       boolAttr(func(w Widgeter, v bool) { w.widgetBase().SetFontBold(v) }),
	"fontitalic":     boolAttr(func(w Widgeter, v bool) { w.widgetBase().SetFontItalic(v) }),
	"fontfamily":     textAttr(func(w Widgeter, v string) { w.widgetBase().SetFontFamily(v) }),
	"accessiblename": textAttr(func(w Widgeter, v string) { w.widgetBase().SetAccessibleName(v) }),
}

// panelXMLAttrs are the attributes of the containers
var panelXMLAttrs = map[string]xmlAttr{
	"padding": intAttr(func(w Widgeter, v int) { w.widgetBase().SetPanelPadding(v) }),
	"spacing": intAttr(func(w Widgeter, v int) { w.widgetBase().SetCellPadding(v) }),
}

// xmlElements are the elements of the XML
var xmlElements map[string]*xmlElement

func init() {
	text := func(set func(w Widgeter, v string)) map[string]xmlAttr {
		return map[string]xmlAttr{"text": textAttr(set)}
	}
	column := &xmlElement{make: func() Widgeter { return NewPanel() }, attrs: panelXMLAttrs, children: "column"}
	row := &xmlElement{make: func() Widgeter { return NewPanel() }, attrs: panelXMLAttrs, children: "row"}
	xmlElements = map[string]*xmlElement{
		"column": column,
		"vbox":   column,
		"panel":  column,
		"row":    row,
		"hbox":   row,
		"frame":  {make: func() Widgeter { return NewFrame() }, attrs: panelXMLAttrs, children: "column"},
		"groupbox": {make: func() Widgeter { return NewGroupBox("") }, children: "column", attrs: merge(panelXMLAttrs, map[string]xmlAttr{
			"title": textAttr(func(w Widgeter, v string) { w.(*GroupBox).SetTitle(v) }),
		})},
		"scrollarea": {make: func() Widgeter { return NewScrollArea() }, children: "column"},
		"hsplitter":  {make: func() Widgeter { return NewHSplitter() }, children: "split"},
		"vsplitter":  {make: func() Widgeter { return NewVSplitter() }, children: "split"},
		"tabwidget":  {make: func() Widgeter { return NewTabWidget() }, children: "tabs"},
		"hspacer":    {make: func() Widgeter { return NewHSpacer() }, children: "none"},
		"vspacer":    {make: func() Widgeter { return NewVSpacer() }, children: "none"},
		"space":      {make: func() Widgeter { return NewSpace() }, children: "none"},
		"label": {make: func() Widgeter { return NewLabel("") }, children: "none",
			attrs: text(func(w Widgeter, v string) { w.(*Label).SetText(v) })},
		"button": {make: func() Widgeter { return NewButton("") }, children: "none", events: []string{"onclick"},
			attrs: text(func(w Widgeter, v string) { w.(*Button).SetText(v) })},
		"link": {make: func() Widgeter { return NewLink("", nil) }, children: "none", attrs: map[string]xmlAttr{
			"text": textAttr(func(w Widgeter, v string) { w.(*Link).SetText(v) }),
			"url":  textAttr(func(w Widgeter, v string) { w.(*Link).SetURL(v) }),
		}},
		"textbox": {make: func() Widgeter { return NewTextBox() }, children: "none", events: []string{"ontextchanged", "onkeydown"},
			attrs: map[string]xmlAttr{
				"text":       textAttr(func(w Widgeter, v string) { w.(*TextBox).SetText(v) }),
				"hint":       textAttr(func(w Widgeter, v string) { w.(*TextBox).SetHint(v) }),
				"multiline":  boolAttr(func(w Widgeter, v bool) { w.(*TextBox).SetMultiline(v) }),
				"readonly":   boolAttr(func(w Widgeter, v bool) { w.(*TextBox).SetReadOnly(v) }),
				"ispassword": boolAttr(func(w Widgeter, v bool) { w.(*TextBox).SetIsPassword(v) }),
			}},
		"checkbox": {make: func() Widgeter { return NewCheckbox("") }, children: "none", events: []string{"onstatechanged"},
			attrs: map[string]xmlAttr{
				"text":     textAttr(func(w Widgeter, v string) { w.(*Checkbox).SetText(v) }),
				"checked":  boolAttr(func(w Widgeter, v bool) { w.(*Checkbox).SetChecked(v) }),
				"tristate": boolAttr(func(w Widgeter, v bool) { w.(*Checkbox).SetTristate(v) }),
			}},
		"radiobutton": {make: func() Widgeter { return NewRadioButton("") }, children: "none", attrs: map[string]xmlAttr{
			"text":    textAttr(func(w Widgeter, v string) { w.(*RadioButton).SetText(v) }),
			"checked": boolAttr(func(w Widgeter, v bool) { w.(*RadioButton).SetChecked(v) }),
		}},
		"toggleswitch": {make: func() Widgeter { return NewToggleSwitch("") }, children: "none", attrs: map[string]xmlAttr{
			"text":    textAttr(func(w Widgeter, v string) { w.(*ToggleSwitch).SetText(v) }),
			"checked": boolAttr(func(w Widgeter, v bool) { w.(*ToggleSwitch).SetChecked(v) }),
		}},
		"combobox": {make: func() Widgeter { return NewComboBox() }, children: "none", attrs: map[string]xmlAttr{
			"items":    textAttr(func(w Widgeter, v string) { w.(*ComboBox).SetItems(splitXMLItems(v)) }),
			"selected": intAttr(func(w Widgeter, v int) { w.(*ComboBox).SetSelectedIndex(v) }),
		}},
		"listview": {make: func() Widgeter { return NewListView() }, children: "none", attrs: map[string]xmlAttr{
			"items":       textAttr(func(w Widgeter, v string) { w.(*ListView).SetItems(splitXMLItems(v)) }),
			"multiselect": boolAttr(func(w Widgeter, v bool) { w.(*ListView).SetMultiselect(v) }),
		}},
		"treeview": {make: func() Widgeter { return NewTreeView() }, children: "none"},
		"table":    {make: func() Widgeter { return NewTable() }, children: "table"},
		"numbox": {make: func() Widgeter { return NewNumBox() }, children: "none", events: []string{"onvaluechanged", "onchanged"},
			attrs: map[string]xmlAttr{
				"min":      floatAttr(func(w Widgeter, v float64) { w.(*NumBox).SetMin(v) }),
				"max":      floatAttr(func(w Widgeter, v float64) { w.(*NumBox).SetMax(v) }),
				"step":     floatAttr(func(w Widgeter, v float64) { w.(*NumBox).SetStep(v) }),
				"decimals": intAttr(func(w Widgeter, v int) { w.(*NumBox).SetDecimals(v) }),
				"value":    floatAttr(func(w Widgeter, v float64) { w.(*NumBox).SetValue(v) }),
			}},
		"slider": {make: func() Widgeter { return NewSlider() }, children: "none", attrs: map[string]xmlAttr{
			"min":   floatAttr(func(w Widgeter, v float64) { w.(*Slider).SetMin(v) }),
			"max":   floatAttr(func(w Widgeter, v float64) { w.(*Slider).SetMax(v) }),
			"step":  floatAttr(func(w Widgeter, v float64) { w.(*Slider).SetStep(v) }),
			"value": floatAttr(func(w Widgeter, v float64) { w.(*Slider).SetValue(v) }),
		}},
		"progressbar": {make: func() Widgeter { return NewProgressBar(0, 100, 0) }, children: "none", attrs: map[string]xmlAttr{
			"value": floatAttr(func(w Widgeter, v float64) { w.(*ProgressBar).SetValue(v) }),
		}},
		"datepicker":  {make: func() Widgeter { return NewDatePicker() }, children: "none"},
		"timepicker":  {make: func() Widgeter { return NewTimePicker() }, children: "none"},
		"colorpicker": {make: func() Widgeter { return NewColorPicker() }, children: "none"},
		"imagebox":    {make: func() Widgeter { return NewImageBox() }, children: "none"},
		// The charts: <linechart values="3,5,4,7" title="Load"/>,
		// <barchart items="Jan:120|Feb:95"/>, <piechart items="Linux:60|Windows:30" donut="0.5"/>
		"linechart": {make: func() Widgeter { return NewLineChart() }, children: "none",
			attrs: chartXMLAttrs("values title xtitle ytitle", func(w Widgeter, k, v string) { w.(*LineChart).ProcessPropChange(k, v) })},
		"barchart": {make: func() Widgeter { return NewBarChart() }, children: "none",
			attrs: chartXMLAttrs("items horizontal stacked title xtitle ytitle", func(w Widgeter, k, v string) { w.(*BarChart).ProcessPropChange(k, v) })},
		"piechart": {make: func() Widgeter { return NewPieChart() }, children: "none",
			attrs: chartXMLAttrs("items donut title", func(w Widgeter, k, v string) { w.(*PieChart).ProcessPropChange(k, v) })},
		// <widget id="chart"/>: a widget made by the application, from the
		// map given to SetLayout
		"widget": {children: "none"},
	}
}

// chartXMLAttrs returns the attributes of a chart element (the names apart
// by spaces), all set by set
func chartXMLAttrs(names string, set func(w Widgeter, key, value string)) map[string]xmlAttr {
	m := map[string]xmlAttr{}
	for _, k := range strings.Fields(names) {
		k := k
		m[k] = textAttr(func(w Widgeter, v string) { set(w, k, v) })
	}
	return m
}

func merge(maps ...map[string]xmlAttr) map[string]xmlAttr {
	all := map[string]xmlAttr{}
	for _, m := range maps {
		for k, v := range m {
			all[k] = v
		}
	}
	return all
}

// splitXMLItems splits items="One|Two|Three"
func splitXMLItems(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "|")
}

// SetLayout builds the widgets of the XML in the widget (see layout_xml.go
// for the elements). handlers holds the functions of the on... attributes:
// a map[string]func(), or a value whose methods of the names are called.
// widgets are the application's own widgets of the <widget id="..."/>
// elements. The error, a *LayoutError, lists the mistakes in the XML; what
// is right is built anyway.
func (c *Widget) SetLayout(layoutAsXml string, handlers interface{}, widgets map[string]Widgeter) error {
	n, err := parseLayoutXML(layoutAsXml)
	if err != nil {
		return fmt.Errorf("layout: the XML: %v", err)
	}
	b := &layoutBuilder{handlers: handlers, widgets: widgets, ids: map[string]int{}}
	b.build(n, c, 0, 0)
	if len(b.problems) > 0 {
		return &LayoutError{Problems: b.problems}
	}
	return nil
}

// build makes the widget of the node in the parent at row, col
func (b *layoutBuilder) build(n *uiNode, parent Widgeter, row, col int) {
	name := n.XMLName.Local
	e, ok := xmlElements[name]
	if !ok {
		b.problem(n, "unknown element")
		return
	}
	var w Widgeter
	id := n.GetAttrValueByName("id", "")
	if id != "" {
		if line, seen := b.ids[id]; seen {
			b.problem(n, "the id %q is used at line %d too", id, line)
		}
		b.ids[id] = n.Line
	}
	if name == "widget" {
		if id == "" {
			b.problem(n, "no id")
			return
		}
		if w, ok = b.widgets[id]; !ok || w == nil {
			b.problem(n, "no widget %q in the widgets given", id)
			return
		}
	} else {
		w = e.make()
		if id != "" {
			w.SetName(id)
		}
	}
	parent.AddWidget(row, col, w)

	for _, attr := range n.Attrs {
		key := attr.Name.Local
		if key == "id" {
			continue
		}
		if strings.HasPrefix(key, "on") {
			if !slices.Contains(e.events, key) && name != "widget" {
				b.problem(n, "no event %s", key)
				continue
			}
			if f := b.handler(n, attr.Value); f != nil {
				w.SetPropFunction(key, f)
			}
			continue
		}
		set, ok := e.attrs[key]
		if !ok {
			set, ok = commonXMLAttrs[key]
		}
		if !ok {
			b.problem(n, "unknown attribute %s", key)
			w.SetProp(key, attr.Value) // a custom widget may read it
			continue
		}
		if err := set(w, attr.Value); err != nil {
			b.problem(n, "%s: %v", key, err)
		}
	}

	switch e.children {
	case "none":
		for i := range n.Nodes {
			b.problem(&n.Nodes[i], "<%s> has no child elements", name)
		}
	case "column", "row":
		for i := range n.Nodes {
			if e.children == "row" {
				b.build(&n.Nodes[i], w, 0, i)
			} else {
				b.build(&n.Nodes[i], w, i, 0)
			}
		}
	case "split":
		if len(n.Nodes) != 2 {
			b.problem(n, "a splitter has two child elements, not %d", len(n.Nodes))
		}
		var panes [2]Widgeter
		for i := 0; i < len(n.Nodes) && i < 2; i++ {
			holder := NewPanel()
			holder.SetPanelPadding(0)
			b.build(&n.Nodes[i], holder, 0, 0)
			panes[i] = holder
		}
		if panes[0] != nil && panes[1] != nil {
			w.(*Splitter).SetWidgets(panes[0], panes[1])
		}
	case "tabs":
		for i := range n.Nodes {
			tab := &n.Nodes[i]
			if tab.XMLName.Local != "tab" {
				b.problem(tab, "a tab widget has <tab> elements only")
				continue
			}
			page := NewPanel()
			for j := range tab.Nodes {
				b.build(&tab.Nodes[j], page, j, 0)
			}
			for _, attr := range tab.Attrs {
				if attr.Name.Local != "text" {
					b.problem(tab, "unknown attribute %s", attr.Name.Local)
				}
			}
			w.(*TabWidget).AddPage(tab.GetAttrValueByName("text", ""), page)
		}
	case "table":
		for i := range n.Nodes {
			if child := n.Nodes[i].XMLName.Local; child != "columns" && child != "rows" {
				b.problem(&n.Nodes[i], "a table has <columns> and <rows> only")
			}
		}
		w.(*Table).TableSetLayoutXml(n)
	}
}

// handler finds the function of an on... attribute
func (b *layoutBuilder) handler(n *uiNode, name string) func() {
	if m, ok := b.handlers.(map[string]func()); ok {
		if f, ok := m[name]; ok {
			return f
		}
		b.problem(n, "no handler %q", name)
		return nil
	}
	if b.handlers == nil {
		b.problem(n, "no handlers given for %q", name)
		return nil
	}
	method := reflect.ValueOf(b.handlers).MethodByName(name)
	if !method.IsValid() {
		b.problem(n, "no method %s", name)
		return nil
	}
	f, ok := method.Interface().(func())
	if !ok {
		b.problem(n, "the method %s is not a func()", name)
		return nil
	}
	return f
}

// Bind puts the widgets of the layout into the fields of the struct that
// target points to: a field of a widget type (*ui.Button, *ui.TextBox...,
// or ui.Widgeter) gets the widget whose id is its tag `ui:"id"`, or else
// its name with the first letter in lower case (BtnOK: "btnOK"). Fields of
// other types, and those tagged `ui:"-"`, are left. The error lists the
// widgets not found and of another type; `ui:"id,optional"` may be missing.
//
//	var w struct {
//		Name   *ui.TextBox
//		OK     *ui.Button `ui:"btnOK"`
//	}
//	panel.SetLayout(layout, handlers, nil)
//	err := ui.Bind(panel, &w)
func Bind(root Widgeter, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return errors.New("bind: the target is not a pointer to a struct")
	}
	s := v.Elem()
	widgeterType := reflect.TypeOf((*Widgeter)(nil)).Elem()
	var problems []string
	for i := range s.NumField() {
		field := s.Type().Field(i)
		if !field.IsExported() || !field.Type.Implements(widgeterType) && field.Type != widgeterType {
			continue
		}
		tag := field.Tag.Get("ui")
		if tag == "-" {
			continue
		}
		id, options, _ := strings.Cut(tag, ",")
		if id == "" {
			id = strings.ToLower(field.Name[:1]) + field.Name[1:]
		}
		w := root.widgetBase().FindWidgetByName(id)
		if w == nil {
			if options != "optional" {
				problems = append(problems, fmt.Sprintf("no widget %q for %s", id, field.Name))
			}
			continue
		}
		if registered, ok := lookupWidget(w.Id()); ok {
			w = registered
		}
		wv := reflect.ValueOf(w)
		if !wv.Type().AssignableTo(field.Type) {
			problems = append(problems, fmt.Sprintf("the widget %q is a %s, not a %s (%s)", id, wv.Type(), field.Type, field.Name))
			continue
		}
		s.Field(i).Set(wv)
	}
	if len(problems) > 0 {
		return errors.New("bind: " + strings.Join(problems, "; "))
	}
	return nil
}
