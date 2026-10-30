package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"
)

// SVGRecorder writes what a canvas draws as SVG: the paths, the fills (the
// gradients and the hatches too), the lines, the rectangles and the texts,
// in logical pixels. Set it on a canvas with Canvas.SetRecorder: the canvas
// draws as usual and the recorder gets the same shapes as vectors, so a
// widget painted into it comes out as a picture that is sharp at any size.
// The images a canvas draws are not recorded.
type SVGRecorder struct {
	width, height float64
	defs          bytes.Buffer
	body          bytes.Buffer
	clips         map[svgClipKey]string
	nextID        int
	lastClip      string
	open          bool
	// paintXf is the transform of the paint being recorded: its
	// coordinates are those of the drawing methods
	paintXf *Matrix
	// layers is the count of the groups of the layers open
	layers int
}

// NewSVGRecorder makes a recorder of a picture of the logical size
func NewSVGRecorder(width, height float64) *SVGRecorder {
	return &SVGRecorder{width: width, height: height, clips: map[svgClipKey]string{}}
}

// SetRecorder makes the canvas record what it draws into the recorder as
// well; nil stops it
func (c *Canvas) SetRecorder(r *SVGRecorder) {
	c.rec = r
}

// WriteTo writes the SVG document
func (r *SVGRecorder) WriteTo(w io.Writer) (int64, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s">`+"\n",
		svgNum(r.width), svgNum(r.height), svgNum(r.width), svgNum(r.height))
	if r.defs.Len() > 0 {
		b.WriteString("<defs>\n")
		b.Write(r.defs.Bytes())
		b.WriteString("</defs>\n")
	}
	b.Write(r.body.Bytes())
	if r.open {
		b.WriteString("</g>\n")
	}
	for i := 0; i < r.layers; i++ {
		b.WriteString("</g>\n") // a layer not ended
	}
	b.WriteString("</svg>\n")
	n, err := w.Write(b.Bytes())
	return int64(n), err
}

// String returns the SVG document
func (r *SVGRecorder) String() string {
	var b strings.Builder
	r.WriteTo(&b)
	return b.String()
}

func (r *SVGRecorder) id(prefix string) string {
	r.nextID++
	return prefix + strconv.Itoa(r.nextID)
}

// clip starts a group of the clip of the canvas, unless the elements before
// are in the same clip
func (r *SVGRecorder) clip(c *Canvas) {
	s := c.scale
	key := svgClipKey{[4]float64{float64(c.state.clipX) / s, float64(c.state.clipY) / s, float64(c.state.clipW) / s, float64(c.state.clipH) / s}, c.state.svgClip}
	id, ok := r.clips[key]
	if !ok {
		id = r.id("c")
		r.clips[key] = id
		box := key.box
		fmt.Fprintf(&r.defs, `<clipPath id="%s"%s><rect x="%s" y="%s" width="%s" height="%s"/></clipPath>`+"\n",
			id, svgClipRef(key.shape), svgNum(box[0]), svgNum(box[1]), svgNum(box[2]), svgNum(box[3]))
	}
	if id == r.lastClip && r.open {
		return
	}
	if r.open {
		r.body.WriteString("</g>\n")
	}
	fmt.Fprintf(&r.body, `<g clip-path="url(#%s)">`+"\n", id)
	r.lastClip, r.open = id, true
}

// svgClipKey is a clip of the canvas: its rectangle and the shape of
// ClipPath within it, if any
type svgClipKey struct {
	box   [4]float64
	shape string
}

// svgClipRef is the attribute that clips an element to the shape of the id
func svgClipRef(id string) string {
	if id == "" {
		return ""
	}
	return ` clip-path="url(#` + id + `)"`
}

// clipPath records the shape of Canvas.ClipPath, within the shapes before
func (r *SVGRecorder) clipPath(c *Canvas, p *Path) {
	id := r.id("s")
	ox, oy := c.origin()
	fmt.Fprintf(&r.defs, `<clipPath id="%s"%s><path d="%s"%s/></clipPath>`+"\n",
		id, svgClipRef(c.state.svgClip), svgPathData(p, ox, oy), svgClipRule(p))
	c.state.svgClip = id
}

// origin returns where the logical point 0, 0 of the canvas is in the picture
func (c *Canvas) origin() (float64, float64) {
	return float64(c.state.translateX) / c.scale, float64(c.state.translateY) / c.scale
}

// svgNum4 writes a number of a matrix, with more digits than a coordinate
func svgNum4(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return strconv.FormatFloat(math.Round(v*1e4)/1e4, 'f', -1, 64)
}

func svgNum(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// svgColor returns the fill or the stroke attributes of the straight color
func svgColor(attr string, col color.Color) string {
	r, g, b, a := straightRGBA8(col)
	s := fmt.Sprintf(`%s="#%02x%02x%02x"`, attr, r, g, b)
	if a < 255 {
		s += fmt.Sprintf(` %s-opacity="%s"`, attr, svgNum(float64(a)/255))
	}
	return s
}

// svgPathData returns the path as the d of an SVG path, moved by (ox, oy)
func svgPathData(p *Path, ox, oy float64) string {
	var b strings.Builder
	pt := func(q PointF) string { return svgNum(q.X+ox) + " " + svgNum(q.Y+oy) }
	for _, op := range p.ops {
		switch op.kind {
		case pathMoveTo:
			b.WriteString("M" + pt(op.p[0]))
		case pathLineTo:
			b.WriteString("L" + pt(op.p[0]))
		case pathQuadTo:
			b.WriteString("Q" + pt(op.p[0]) + " " + pt(op.p[1]))
		case pathCubicTo:
			b.WriteString("C" + pt(op.p[0]) + " " + pt(op.p[1]) + " " + pt(op.p[2]))
		case pathArc:
			arcCubics(op, func(c1, c2, e PointF) {
				b.WriteString("C" + pt(c1) + " " + pt(c2) + " " + pt(e))
			})
		case pathClose:
			b.WriteString("Z")
		}
	}
	return b.String()
}

func (r *SVGRecorder) fillPath(c *Canvas, p *Path, col color.Color) {
	r.clip(c)
	ox, oy := c.origin()
	fmt.Fprintf(&r.body, `<path d="%s" %s%s/>`+"\n", svgPathData(p, ox, oy), svgColor("fill", col), svgFillRule(p))
}

func (r *SVGRecorder) fillPathPaint(c *Canvas, p *Path, paint Paint) {
	r.clip(c)
	ox, oy := c.origin()
	fill, ok := r.paintAttr(c, "fill", p, paint)
	if !ok {
		return
	}
	fmt.Fprintf(&r.body, `<path d="%s" %s%s/>`+"\n", svgPathData(p, ox, oy), fill, svgFillRule(p))
}

// paintAttr returns the attribute (fill or stroke) that paints the path
// with the paint, its gradient or pattern added to the definitions
func (r *SVGRecorder) paintAttr(c *Canvas, attr string, p *Path, paint Paint) (string, bool) {
	ox, oy := c.origin()
	fill := ""
	// The coordinates of a paint under a transform stay its own; the
	// gradient or the pattern is transformed to the picture
	gx, gy, gxf := ox, oy, ""
	if r.paintXf != nil {
		m := r.paintXf.Then(TranslationMatrix(ox, oy))
		gx, gy = 0, 0
		gxf = fmt.Sprintf(`matrix(%s %s %s %s %s %s)`, svgNum4(m.A), svgNum4(m.B), svgNum4(m.C), svgNum4(m.D), svgNum(m.E), svgNum(m.F))
	}
	gradXf := ""
	if gxf != "" {
		gradXf = ` gradientTransform="` + gxf + `"`
	}
	spread := func(s GradientSpread) string {
		switch s {
		case SpreadRepeat:
			return ` spreadMethod="repeat"`
		case SpreadReflect:
			return ` spreadMethod="reflect"`
		}
		return ""
	}
	stops := func(st []GradientStop) string {
		var b strings.Builder
		for _, s := range st {
			fmt.Fprintf(&b, `<stop offset="%s" stop-color="#%02x%02x%02x" stop-opacity="%s"/>`,
				svgNum(s.Offset), s.Color.R, s.Color.G, s.Color.B, svgNum(float64(s.Color.A)/255))
		}
		return b.String()
	}
	switch g := paint.(type) {
	case LinearGradient:
		id := r.id("g")
		fmt.Fprintf(&r.defs, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s"%s>%s</linearGradient>`+"\n",
			id, svgNum(g.X1+gx), svgNum(g.Y1+gy), svgNum(g.X2+gx), svgNum(g.Y2+gy), gradXf+spread(g.Spread), stops(gradientStops(g.Stops, g.LinearRGB)))
		fill = attr + `="url(#` + id + `)"`
	case RadialGradient:
		id := r.id("g")
		fmt.Fprintf(&r.defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s"%s>%s</radialGradient>`+"\n",
			id, svgNum(g.CX+gx), svgNum(g.CY+gy), svgNum(g.R), gradXf+spread(g.Spread), stops(gradientStops(g.Stops, g.LinearRGB)))
		fill = attr + `="url(#` + id + `)"`
	case solidPaint:
		fill = svgColor(attr, color.RGBA(g))
	case twoCircleGradient:
		// The circles of the gradient: the focal one at 0, the outer at 1
		id := r.id("g")
		fmt.Fprintf(&r.defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s" fx="%s" fy="%s" fr="%s"%s>%s</radialGradient>`+"\n",
			id, svgNum(g.x1+gx), svgNum(g.y1+gy), svgNum(g.r1), svgNum(g.x0+gx), svgNum(g.y0+gy), svgNum(g.r0), gradXf+spread(g.spread), stops(g.stops))
		fill = attr + `="url(#` + id + `)"`
	case HatchPaint:
		id := r.id("h")
		spacing, width := g.Spacing, g.Width
		if spacing <= 0 {
			spacing = 6
		}
		if width <= 0 {
			width = 1
		}
		bg := ""
		if g.Background.A > 0 {
			bg = fmt.Sprintf(`<rect width="%s" height="%s" %s/>`, svgNum(spacing), svgNum(spacing), svgColor("fill", g.Background))
		}
		// Horizontal stripes turned by the angle: up to the right on the screen
		fmt.Fprintf(&r.defs, `<pattern id="%s" patternUnits="userSpaceOnUse" width="%s" height="%s" patternTransform="%srotate(%s)">%s<rect y="%s" width="%s" height="%s" %s/></pattern>`+"\n",
			id, svgNum(spacing), svgNum(spacing), gxf+" ", svgNum(-g.Angle*180/math.Pi), bg,
			svgNum((spacing-width)/2), svgNum(spacing), svgNum(width), svgColor("fill", g.Color))
		fill = attr + `="url(#` + id + `)"`
	default:
		// A paint of the application's own: its color in the middle
		minX, minY, maxX, maxY, ok := p.Bounds()
		if !ok {
			return "", false
		}
		fill = svgColor(attr, paint.ColorAt((minX+maxX)/2, (minY+maxY)/2))
	}
	return fill, true
}

// strokePath records the outline of the path in the color, or the paint
// if it is not nil
func (r *SVGRecorder) strokePath(c *Canvas, p *Path, style StrokeStyle, col color.Color, paint Paint) {
	r.clip(c)
	ox, oy := c.origin()
	w := style.Width
	if w <= 0 {
		w = 1
	}
	stroke := ""
	if paint != nil {
		var ok bool
		if stroke, ok = r.paintAttr(c, "stroke", p, paint); !ok {
			return
		}
	} else {
		stroke = svgColor("stroke", col)
	}
	attrs := []string{stroke, `fill="none"`, `stroke-width="` + svgNum(w) + `"`}
	switch style.Join {
	case LineJoinRound:
		attrs = append(attrs, `stroke-linejoin="round"`)
	case LineJoinBevel:
		attrs = append(attrs, `stroke-linejoin="bevel"`)
	default:
		if style.MiterLimit > 0 {
			attrs = append(attrs, `stroke-miterlimit="`+svgNum(style.MiterLimit)+`"`)
		}
	}
	switch style.Cap {
	case LineCapRound:
		attrs = append(attrs, `stroke-linecap="round"`)
	case LineCapSquare:
		attrs = append(attrs, `stroke-linecap="square"`)
	}
	if len(style.Dash) > 0 {
		parts := make([]string, len(style.Dash))
		for i, d := range style.Dash {
			parts[i] = svgNum(d)
		}
		attrs = append(attrs, `stroke-dasharray="`+strings.Join(parts, " ")+`"`)
		if style.DashOffset != 0 {
			attrs = append(attrs, `stroke-dashoffset="`+svgNum(style.DashOffset)+`"`)
		}
	}
	fmt.Fprintf(&r.body, `<path d="%s" %s/>`+"\n", svgPathData(p, ox, oy), strings.Join(attrs, " "))
}

func (r *SVGRecorder) rect(c *Canvas, x, y, w, h float64, col color.Color, stroke bool) {
	r.clip(c)
	ox, oy := c.origin()
	paint := svgColor("fill", col)
	if stroke {
		paint = svgColor("stroke", col) + ` fill="none" stroke-width="1"`
		x, y, w, h = x+0.5, y+0.5, w-1, h-1
	}
	fmt.Fprintf(&r.body, `<rect x="%s" y="%s" width="%s" height="%s" %s/>`+"\n",
		svgNum(x+ox), svgNum(y+oy), svgNum(w), svgNum(h), paint)
}

// svgFont returns the font attributes of the canvas' font
func (c *Canvas) svgFont() string {
	family := "sans-serif"
	fam, style := c.state.fontFamily, ""
	if i := strings.Index(fam, ":"); i >= 0 {
		fam, style = fam[:i], fam[i+1:]
	}
	if strings.Contains(strings.ToLower(fam), "mono") {
		family = "monospace"
	} else if fam != "" && fam != FontFamilySans {
		family = `'` + strings.ReplaceAll(fam, `'`, ``) + `', sans-serif`
	}
	s := fmt.Sprintf(`font-family="%s" font-size="%s"`, family, svgNum(c.state.fontSize))
	if strings.Contains(style, "bold") {
		s += ` font-weight="bold"`
	}
	if strings.Contains(style, "italic") {
		s += ` font-style="italic"`
	}
	return s
}

func svgEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// text records a line of text with its top-left corner at (x, y) of the
// canvas, turned by the angle around the anchor (ax, ay)
func (r *SVGRecorder) text(c *Canvas, text string, x, y, lineH, angle, ax, ay float64) {
	r.clip(c)
	ox, oy := c.origin()
	// The baseline about four fifths down the line
	tr := ""
	if angle != 0 {
		tr = fmt.Sprintf(` transform="rotate(%s %s %s)"`, svgNum(angle*180/math.Pi), svgNum(ax+ox), svgNum(ay+oy))
	}
	fmt.Fprintf(&r.body, `<text x="%s" y="%s" %s %s%s>%s</text>`+"\n",
		svgNum(x+ox), svgNum(y+oy+lineH*0.78), c.svgFont(), svgColor("fill", c.state.col), tr, svgEscape(text))
}

// svgFillRule returns the fill-rule attribute of the path, if it is not
// the default
func svgFillRule(p *Path) string {
	if p.fillRule == FillRuleEvenOdd {
		return ` fill-rule="evenodd"`
	}
	return ""
}

func svgClipRule(p *Path) string {
	if p.fillRule == FillRuleEvenOdd {
		return ` clip-rule="evenodd"`
	}
	return ""
}

// svgBlendModes are the blend modes as CSS names them
var svgBlendModes = map[BlendMode]string{
	BlendMultiply: "multiply", BlendScreen: "screen", BlendOverlay: "overlay", BlendDarken: "darken",
	BlendLighten: "lighten", BlendColorDodge: "color-dodge", BlendColorBurn: "color-burn",
	BlendHardLight: "hard-light", BlendSoftLight: "soft-light", BlendDifference: "difference",
	BlendExclusion: "exclusion", BlendHue: "hue", BlendSaturation: "saturation", BlendColor: "color",
	BlendLuminosity: "luminosity", BlendAdd: "plus-lighter",
}

// closeClip ends the group of the clip open, so that a group of a layer
// nests around whole groups
func (r *SVGRecorder) closeClip() {
	if r.open {
		r.body.WriteString("</g>\n")
		r.open, r.lastClip = false, ""
	}
}

// beginLayer starts the group of a layer of BeginLayerStyle: its opacity,
// its blend, its mask. It returns false for a blend SVG has not (BlendErase,
// BlendKeepInside, BlendAtop): such a layer is not recorded.
func (r *SVGRecorder) beginLayer(c *Canvas, st LayerStyle) bool {
	attrs := ""
	style := ""
	switch st.Blend {
	case BlendNormal:
	case BlendErase, BlendKeepInside, BlendAtop:
		return false
	default:
		style = "mix-blend-mode:" + svgBlendModes[st.Blend] + ";"
	}
	if st.Isolated || st.Blend != BlendNormal || st.Mask != nil {
		style += "isolation:isolate;"
	}
	if st.Opacity > 0 && st.Opacity < 1 {
		attrs += ` opacity="` + svgNum4(st.Opacity) + `"`
	}
	if style != "" {
		attrs += ` style="` + style + `"`
	}
	if st.Mask != nil {
		// The mask: the area of the layer painted with the paint, its alpha
		// the opacity
		s := c.scale
		ox, oy := c.origin()
		area := NewPath()
		area.Rect(float64(c.state.clipX)/s-ox, float64(c.state.clipY)/s-oy, float64(c.state.clipW)/s, float64(c.state.clipH)/s)
		if c.state.hasXf {
			xf := c.state.xf
			r.paintXf = &xf
		}
		fill, ok := r.paintAttr(c, "fill", area, st.Mask)
		r.paintXf = nil
		if ok {
			id := r.id("m")
			fmt.Fprintf(&r.defs, `<mask id="%s" mask-type="alpha" maskUnits="userSpaceOnUse"><path d="%s" %s/></mask>`+"\n",
				id, svgPathData(area, ox, oy), fill)
			attrs += ` mask="url(#` + id + `)"`
		}
	}
	r.closeClip()
	r.body.WriteString("<g" + attrs + ">\n")
	r.layers++
	return true
}

// endLayer ends the group of the last layer
func (r *SVGRecorder) endLayer() {
	if r.layers == 0 {
		return
	}
	r.closeClip()
	r.body.WriteString("</g>\n")
	r.layers--
}
