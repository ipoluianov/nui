package ui

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// PropertyGrid is a list of named values - properties - edited in place,
// grouped in collapsible categories. Each property has an editor for its
// type: a text box, a number box, a checkbox, a combo box, a color picker or
// a date picker, or any widget (AddCustom).
//
//	grid := ui.NewPropertyGrid()
//	grid.AddString("General", "Name", "Server 1")
//	grid.AddInt("Network", "Port", 8080, 1, 65535)
//	grid.AddBool("Network", "Use TLS", true)
//	grid.SetOnChanged(func(p *ui.Property) { fmt.Println(p.Name(), p.Value()) })
//
// SetObject fills the grid from the exported fields of a struct and writes
// the edits back into it (see SetObject for the field tags).
type PropertyGrid struct {
	Widget
	sections []*propertySection
	props    []*Property
	spacer   *VSpacer

	nameWidth int
	onChanged func(p *Property)
}

// propertySection is a category: an expander with a two-column grid (names
// and editors) in it. The properties without a category are in a plain panel
// at the top.
type propertySection struct {
	name     string
	expander *Expander
	panel    *Panel
	rows     int
}

const (
	propertyGridMinNameWidth = 80
	propertyGridMaxNameWidth = 260
)

func NewPropertyGrid() *PropertyGrid {
	var c PropertyGrid
	c.InitWidget()
	c.SetTypeName("PropertyGrid")
	c.SetPanelPadding(4)
	c.SetCellPadding(4)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetAllowScroll(false, true)
	c.nameWidth = propertyGridMinNameWidth
	c.spacer = NewVSpacer()
	c.Widget.AddWidget(0, 0, c.spacer)
	return &c
}

// SetOnChanged sets the function called when the user changes a property
func (c *PropertyGrid) SetOnChanged(f func(p *Property)) {
	c.onChanged = f
}

// Properties returns the properties in the order they were added
func (c *PropertyGrid) Properties() []*Property {
	return c.props
}

// Property returns the property with the name, nil if there is none
func (c *PropertyGrid) Property(name string) *Property {
	for _, p := range c.props {
		if p.name == name {
			return p
		}
	}
	return nil
}

// Category returns the expander of the category, nil if there is none, e.g.
// to collapse it
func (c *PropertyGrid) Category(name string) *Expander {
	for _, s := range c.sections {
		if s.name == name {
			return s.expander
		}
	}
	return nil
}

// Clear removes all the properties
func (c *PropertyGrid) Clear() {
	for _, s := range c.sections {
		if s.expander != nil {
			c.Widget.RemoveWidget(s.expander)
		} else {
			c.Widget.RemoveWidget(s.panel)
		}
	}
	c.sections = nil
	c.props = nil
	c.nameWidth = propertyGridMinNameWidth
	c.placeSpacer()
	c.form.UpdateLayout()
}

func (c *PropertyGrid) section(category string) *propertySection {
	for _, s := range c.sections {
		if s.name == category {
			return s
		}
	}
	s := &propertySection{name: category}
	if category == "" {
		s.panel = NewPanel()
		s.panel.SetPanelPadding(0)
		s.panel.SetCellPadding(4)
		// The properties without a category come first
		c.sections = append([]*propertySection{s}, c.sections...)
	} else {
		s.expander = NewExpander(category)
		s.expander.SetExpanded(true)
		s.panel = s.expander.Content()
		s.panel.SetCellPadding(4)
		c.sections = append(c.sections, s)
	}
	for i, section := range c.sections {
		var w Widgeter = section.panel
		if section.expander != nil {
			w = section.expander
		}
		if section == s {
			c.Widget.AddWidget(i, 0, w)
		} else {
			w.SetGridPosition(i, 0)
		}
	}
	c.placeSpacer()
	return s
}

// placeSpacer keeps the spacer below the sections, taking the free space
func (c *PropertyGrid) placeSpacer() {
	c.spacer.SetGridPosition(len(c.sections), 0)
	c.ClearLayoutCache()
	c.updateLayout(c.w, c.h, c.w, c.h)
}

// propertyNameLabel is a property's name. Its width is the grid's name
// column width, the same in all the categories, so the editors line up.
type propertyNameLabel struct {
	*Label
	grid *PropertyGrid
}

func (l propertyNameLabel) MinWidth() int {
	return l.grid.nameWidth
}

func (l propertyNameLabel) MaxWidth() int {
	return l.grid.nameWidth
}

func (c *PropertyGrid) add(category, name string, editor Widgeter) *Property {
	s := c.section(category)
	p := &Property{grid: c, name: name, category: category, editor: editor}
	p.label = propertyNameLabel{NewLabel(name), c}
	s.panel.AddWidget(s.rows, 0, p.label)
	s.panel.AddWidget(s.rows, 1, editor)
	s.rows++
	c.props = append(c.props, p)
	c.updateNameWidth(name)
	c.form.UpdateLayout()
	return p
}

// updateNameWidth widens the names column of all the categories to fit the
// name (see propertyNameLabel)
func (c *PropertyGrid) updateNameWidth(name string) {
	w, _, err := MeasureText(c.FontFamily(), c.FontSize(), name)
	if err == nil {
		c.nameWidth = max(c.nameWidth, min(w+12, propertyGridMaxNameWidth))
	}
	c.ClearLayoutCache()
}

// AddString adds a property edited in a text box
func (c *PropertyGrid) AddString(category, name, value string) *Property {
	box := NewTextBox()
	box.SetText(value)
	p := c.add(category, name, box)
	p.get = func() any { return box.Text() }
	p.set = func(v any) { box.SetText(fmt.Sprint(v)) }
	box.SetOnTextChanged(p.changed)
	return p
}

// AddInt adds a whole number property edited in a number box, from min to max
func (c *PropertyGrid) AddInt(category, name string, value, min, max int) *Property {
	return c.addWhole(category, name, float64(value), float64(min), float64(max))
}

// addWhole is AddInt with the value and the limits as float64: the limits of
// a 64-bit field don't fit an int once converted to float64 (2^63-1 becomes
// 2^63), nor do the values of a uint64 above the int range
func (c *PropertyGrid) addWhole(category, name string, value, min, max float64) *Property {
	box := NewNumBox()
	box.SetDecimals(0)
	box.SetMin(min)
	box.SetMax(max)
	box.SetValue(value)
	p := c.add(category, name, box)
	p.get = func() any { return int(saturateInt64(box.Value())) }
	p.set = func(v any) {
		if f, ok := toFloat64(v); ok {
			box.SetValue(f)
		}
	}
	box.SetOnValueChanged(p.changed)
	return p
}

// AddFloat adds a number property edited in a number box with the decimals
func (c *PropertyGrid) AddFloat(category, name string, value float64, decimals int) *Property {
	box := NewNumBox()
	box.SetDecimals(decimals)
	box.SetMin(-math.MaxFloat64)
	box.SetMax(math.MaxFloat64)
	box.SetValue(value)
	p := c.add(category, name, box)
	p.get = func() any { return box.Value() }
	p.set = func(v any) {
		if f, ok := toFloat64(v); ok {
			box.SetValue(f)
		}
	}
	box.SetOnValueChanged(p.changed)
	return p
}

// AddBool adds a yes/no property edited in a checkbox
func (c *PropertyGrid) AddBool(category, name string, value bool) *Property {
	box := NewCheckbox("")
	box.SetMinWidth(ThemeControlHeight())
	box.SetChecked(value)
	p := c.add(category, name, box)
	p.get = func() any { return box.Checked() }
	p.set = func(v any) {
		if b, ok := v.(bool); ok {
			box.SetChecked(b)
		}
	}
	box.SetOnStateChanged(p.changed)
	return p
}

// AddChoice adds a property chosen from the options in a combo box. Its value
// is the index of the chosen option; SetValue also takes the option's text.
func (c *PropertyGrid) AddChoice(category, name string, options []string, index int) *Property {
	box := NewComboBox()
	for _, option := range options {
		box.AddItem(option, option)
	}
	box.SetSelectedIndex(index)
	p := c.add(category, name, box)
	p.get = func() any { return box.SelectedIndex() }
	p.set = func(v any) {
		if text, ok := v.(string); ok {
			for i, option := range options {
				if option == text {
					box.SetSelectedIndex(i)
				}
			}
			return
		}
		if f, ok := toFloat64(v); ok {
			box.SetSelectedIndex(int(f))
		}
	}
	box.SetOnSelectedIndexChanged(p.changed)
	return p
}

// AddColor adds a color property edited in a color picker
func (c *PropertyGrid) AddColor(category, name string, value color.RGBA) *Property {
	picker := NewColorPicker()
	picker.SetColor(value)
	p := c.add(category, name, picker)
	p.get = func() any { return picker.Color() }
	p.set = func(v any) {
		if col, ok := v.(color.Color); ok {
			picker.SetColor(col)
		}
	}
	picker.SetOnColorChanged(func(color.RGBA) { p.changed() })
	return p
}

// AddDate adds a date property edited in a date picker
func (c *PropertyGrid) AddDate(category, name string, value time.Time) *Property {
	picker := NewDatePicker()
	picker.SetDate(value)
	p := c.add(category, name, picker)
	p.get = func() any { return picker.Date() }
	p.set = func(v any) {
		if t, ok := v.(time.Time); ok {
			picker.SetDate(t)
		}
	}
	picker.SetOnDateChanged(func(time.Time) { p.changed() })
	return p
}

// AddCustom adds a property edited in any widget. Its Value is nil: read the
// widget itself, and call the property's NotifyChanged when it changes.
func (c *PropertyGrid) AddCustom(category, name string, editor Widgeter) *Property {
	return c.add(category, name, editor)
}

// Property is one row of a PropertyGrid: a name and the editor of its value.
type Property struct {
	grid     *PropertyGrid
	name     string
	category string
	label    propertyNameLabel
	editor   Widgeter

	get func() any
	set func(v any)
	// field is the struct field the property edits, see SetObject
	field reflect.Value
	// updating is set while the value is set from the code: the editors
	// report those changes too, but they aren't the user's
	updating  bool
	onChanged func()
}

func (p *Property) Name() string {
	return p.name
}

func (p *Property) Category() string {
	return p.category
}

// Editor returns the widget that edits the value
func (p *Property) Editor() Widgeter {
	return p.editor
}

// Value returns the value: string, int, float64, bool, the option index
// (int), color.RGBA or time.Time, depending on how the property was added
func (p *Property) Value() any {
	if p.get == nil {
		return nil
	}
	return p.get()
}

// SetValue sets the value without calling the change functions
func (p *Property) SetValue(v any) {
	if p.set == nil {
		return
	}
	p.updating = true
	p.set(v)
	p.updating = false
}

// SetDescription sets the text shown when the mouse rests on the property
func (p *Property) SetDescription(text string) {
	p.label.SetTooltip(text)
	p.editor.SetProp("tooltip", text)
}

// SetReadOnly shows the value without letting the user change it
func (p *Property) SetReadOnly(readOnly bool) {
	if box, ok := p.editor.(*TextBox); ok {
		box.SetReadOnly(readOnly)
		return
	}
	p.editor.SetEnabled(!readOnly)
}

// SetOnChanged sets the function called when the user changes this property
func (p *Property) SetOnChanged(f func()) {
	p.onChanged = f
}

// NotifyChanged reports a change made by the user in a custom editor (see AddCustom)
func (p *Property) NotifyChanged() {
	p.changed()
}

func (p *Property) changed() {
	if p.updating {
		return
	}
	p.writeField()
	if p.onChanged != nil {
		p.onChanged()
	}
	if p.grid.onChanged != nil {
		p.grid.onChanged(p)
	}
}

// SetObject fills the grid with the exported fields of the struct obj points
// to, and writes the user's changes back into those fields. Supported field
// types: string, bool, the integer and float types, color.RGBA and time.Time.
// Field tags:
//
//	prop:"Display name"    the name shown ("-" skips the field)
//	category:"Network"     the category
//	desc:"..."             the tooltip
//	min:"0" max:"100"      the range of a number
//	decimals:"2"           the decimals of a float
//	options:"Low,High"     a string (or int index) chosen from the options
//	readonly:"true"        shown but not editable
//
// Call Refresh after changing the struct from the code.
func (c *PropertyGrid) SetObject(obj any) error {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("PropertyGrid.SetObject: want a pointer to a struct, got %T", obj)
	}
	c.Clear()
	v = v.Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("prop")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		p := c.addField(f, v.Field(i), name, f.Tag.Get("category"))
		if p == nil {
			continue
		}
		p.field = v.Field(i)
		if desc := f.Tag.Get("desc"); desc != "" {
			p.SetDescription(desc)
		}
		if f.Tag.Get("readonly") == "true" {
			p.SetReadOnly(true)
		}
	}
	return nil
}

var (
	colorRGBAType = reflect.TypeOf(color.RGBA{})
	timeType      = reflect.TypeOf(time.Time{})
)

func (c *PropertyGrid) addField(f reflect.StructField, fv reflect.Value, name, category string) *Property {
	tagFloat := func(key string, def float64) float64 {
		if s := f.Tag.Get(key); s != "" {
			if x, err := strconv.ParseFloat(s, 64); err == nil {
				return x
			}
		}
		return def
	}

	if options := f.Tag.Get("options"); options != "" {
		list := strings.Split(options, ",")
		for i := range list {
			list[i] = strings.TrimSpace(list[i])
		}
		switch fv.Kind() {
		case reflect.String:
			index := 0
			for i, option := range list {
				if option == fv.String() {
					index = i
				}
			}
			return c.AddChoice(category, name, list, index)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return c.AddChoice(category, name, list, int(fv.Int()))
		}
	}

	switch {
	case fv.Type() == colorRGBAType:
		return c.AddColor(category, name, fv.Interface().(color.RGBA))
	case fv.Type() == timeType:
		return c.AddDate(category, name, fv.Interface().(time.Time))
	}

	switch fv.Kind() {
	case reflect.String:
		return c.AddString(category, name, fv.String())
	case reflect.Bool:
		return c.AddBool(category, name, fv.Bool())
	// Without min/max tags a whole number can take any value of its type.
	// (The limits used to be ±1e9: a larger value showed as 1e9, and any edit
	// wrote that into the object.)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		lo := max(tagFloat("min", math.Inf(-1)), float64(minIntOf(fv.Type())))
		hi := min(tagFloat("max", math.Inf(1)), float64(maxIntOf(fv.Type())))
		return c.addWhole(category, name, float64(fv.Int()), lo, hi)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		lo := max(tagFloat("min", 0), 0)
		hi := min(tagFloat("max", math.Inf(1)), float64(maxUintOf(fv.Type())))
		return c.addWhole(category, name, float64(fv.Uint()), lo, hi)
	case reflect.Float32, reflect.Float64:
		p := c.AddFloat(category, name, fv.Float(), int(tagFloat("decimals", 2)))
		box := p.editor.(*NumBox)
		box.SetMin(tagFloat("min", -math.MaxFloat64))
		box.SetMax(tagFloat("max", math.MaxFloat64))
		return p
	}
	return nil
}

func minIntOf(t reflect.Type) int64 {
	return -1 << (t.Bits() - 1)
}

func maxIntOf(t reflect.Type) int64 {
	return 1<<(t.Bits()-1) - 1
}

func maxUintOf(t reflect.Type) uint64 {
	if t.Bits() == 64 {
		return math.MaxUint64
	}
	return 1<<t.Bits() - 1
}

// writeField stores the value in the struct field the property edits
func (p *Property) writeField() {
	if !p.field.IsValid() || !p.field.CanSet() {
		return
	}
	v := p.Value()
	fv := p.field
	switch fv.Kind() {
	case reflect.String:
		if box, ok := p.editor.(*ComboBox); ok {
			fv.SetString(box.SelectedItemText())
		} else {
			fv.SetString(fmt.Sprint(v))
		}
		return
	case reflect.Bool:
		fv.SetBool(v.(bool))
		return
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if f, ok := p.wholeValue(v); ok {
			fv.SetInt(saturateInt64(f))
		}
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if f, ok := p.wholeValue(v); ok && f >= 0 {
			fv.SetUint(saturateUint64(f))
		}
		return
	case reflect.Float32, reflect.Float64:
		if f, ok := toFloat64(v); ok {
			fv.SetFloat(f)
		}
		return
	}
	if fv.Type() == timeType {
		if t, ok := v.(time.Time); ok {
			fv.Set(reflect.ValueOf(withDateOf(fv.Interface().(time.Time), t)))
		}
		return
	}
	if rv := reflect.ValueOf(v); rv.IsValid() && rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)
	}
}

// wholeValue is the number of a whole number property: read from its number
// box, not through Value's int, which a uint64 above the int range overflows
func (p *Property) wholeValue(v any) (float64, bool) {
	if box, ok := p.editor.(*NumBox); ok {
		return box.Value(), true
	}
	return toFloat64(v)
}

// withDateOf is old with the date of picked: the date picker edits the date
// only, the time of day and the location of the field stay. (The field used
// to get midnight in the local time zone.) A zero time takes picked as is.
func withDateOf(old, picked time.Time) time.Time {
	if old.IsZero() {
		return picked
	}
	y, m, d := picked.Date()
	return time.Date(y, m, d, old.Hour(), old.Minute(), old.Second(), old.Nanosecond(), old.Location())
}

// saturateInt64 rounds f to an int64, the out of range values to its limits
func saturateInt64(f float64) int64 {
	f = math.Round(f)
	switch {
	case f >= math.MaxInt64: // 2^63 as a float64
		return math.MaxInt64
	case f <= math.MinInt64:
		return math.MinInt64
	}
	return int64(f)
}

// saturateUint64 rounds f to a uint64, the out of range values to its limits
func saturateUint64(f float64) uint64 {
	f = math.Round(f)
	switch {
	case f <= 0:
		return 0
	case f >= math.MaxUint64: // 2^64 as a float64
		return math.MaxUint64
	}
	return uint64(f)
}

// Refresh shows the current values of the struct given to SetObject, after
// the code changed them
func (c *PropertyGrid) Refresh() {
	for _, p := range c.props {
		if !p.field.IsValid() {
			continue
		}
		fv := p.field
		switch fv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			p.SetValue(float64(fv.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			p.SetValue(float64(fv.Uint()))
		default:
			p.SetValue(fv.Interface())
		}
	}
}

// toFloat64 converts a number of any type
func toFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}
