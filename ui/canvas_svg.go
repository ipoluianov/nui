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
	clips         map[[4]float64]string
	nextID        int
	lastClip      string
	open          bool
}

// NewSVGRecorder makes a recorder of a picture of the logical size
func NewSVGRecorder(width, height float64) *SVGRecorder {
	return &SVGRecorder{width: width, height: height, clips: map[[4]float64]string{}}
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
	box := [4]float64{float64(c.state.clipX) / s, float64(c.state.clipY) / s, float64(c.state.clipW) / s, float64(c.state.clipH) / s}
	id, ok := r.clips[box]
	if !ok {
		id = r.id("c")
		r.clips[box] = id
		fmt.Fprintf(&r.defs, `<clipPath id="%s"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath>`+"\n",
			id, svgNum(box[0]), svgNum(box[1]), svgNum(box[2]), svgNum(box[3]))
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

// origin returns where the logical point 0, 0 of the canvas is in the picture
func (c *Canvas) origin() (float64, float64) {
	return float64(c.state.translateX) / c.scale, float64(c.state.translateY) / c.scale
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
		case pathClose:
			b.WriteString("Z")
		}
	}
	return b.String()
}

func (r *SVGRecorder) fillPath(c *Canvas, p *Path, col color.Color) {
	r.clip(c)
	ox, oy := c.origin()
	fmt.Fprintf(&r.body, `<path d="%s" %s/>`+"\n", svgPathData(p, ox, oy), svgColor("fill", col))
}

func (r *SVGRecorder) fillPathPaint(c *Canvas, p *Path, paint Paint) {
	r.clip(c)
	ox, oy := c.origin()
	fill := ""
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
		fmt.Fprintf(&r.defs, `<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s">%s</linearGradient>`+"\n",
			id, svgNum(g.X1+ox), svgNum(g.Y1+oy), svgNum(g.X2+ox), svgNum(g.Y2+oy), stops(g.Stops))
		fill = `fill="url(#` + id + `)"`
	case RadialGradient:
		id := r.id("g")
		fmt.Fprintf(&r.defs, `<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%s" cy="%s" r="%s">%s</radialGradient>`+"\n",
			id, svgNum(g.CX+ox), svgNum(g.CY+oy), svgNum(g.R), stops(g.Stops))
		fill = `fill="url(#` + id + `)"`
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
		fmt.Fprintf(&r.defs, `<pattern id="%s" patternUnits="userSpaceOnUse" width="%s" height="%s" patternTransform="rotate(%s)">%s<rect y="%s" width="%s" height="%s" %s/></pattern>`+"\n",
			id, svgNum(spacing), svgNum(spacing), svgNum(-g.Angle*180/math.Pi), bg,
			svgNum((spacing-width)/2), svgNum(spacing), svgNum(width), svgColor("fill", g.Color))
		fill = `fill="url(#` + id + `)"`
	default:
		// A paint of the application's own: its color in the middle
		minX, minY, maxX, maxY, ok := p.Bounds()
		if !ok {
			return
		}
		fill = svgColor("fill", paint.ColorAt((minX+maxX)/2, (minY+maxY)/2))
	}
	fmt.Fprintf(&r.body, `<path d="%s" %s/>`+"\n", svgPathData(p, ox, oy), fill)
}

func (r *SVGRecorder) strokePath(c *Canvas, p *Path, style StrokeStyle, col color.Color) {
	r.clip(c)
	ox, oy := c.origin()
	w := style.Width
	if w <= 0 {
		w = 1
	}
	attrs := []string{svgColor("stroke", col), `fill="none"`, `stroke-width="` + svgNum(w) + `"`}
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
