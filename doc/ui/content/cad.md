# Building CAD programs with nui

nui draws what a CAD program shows: drawings of any size zoomed from the
whole sheet to a micrometer, with lines that keep their width, exact arcs,
text of the drawing, hatches and dimensions; and parts in 3D, with the
edges of a drawing and section views. The geometry the program edits - its
entities, constraints, files - is the program's own; nui gives the views.

Run `go run . cad` for an example: a drawing of a flange plate with layers,
snaps, tools, box selection and export, and the part in 3D cut by a plane
(`examples/cad`).

## What nui gives

| Need | nui |
|---|---|
| a view of a drawing: zoom to the pointer, pan, zoom to all, zoom to a box | [View2D](view2d.md) |
| coordinates in millimeters, y up, `float64` throughout | `View2D`, `Matrix`, `Path` |
| lines as thick at any zoom, linetypes | `StrokeStyle{NonScaling: true, Dash: ...}` |
| exact arcs and circles at any zoom | `Path.Arc`, `Circle`, `Ellipse` ([canvas.md](canvas.md)) |
| text of the drawing: sharp, turned, in SVG | `Path.Text`, `View2D.AddText` |
| hatches in drawing units | `FillPathPaint` with `HatchPaint` under the view's transform |
| arrows, markers of snaps, grips | `Path`, `DrawMarker`, `View2DPaint.InScreen` |
| the shapes in view, under the pointer, in a box | `SpatialIndex`, `RectF` |
| crosshair, pick box, rubber band, window and crossing selection | `View2D` (`SetCrosshair`, `SetBoxSelection`, `SetOverlay`) |
| a cross and a move cursor | `MouseCursorCross`, `MouseCursorMove` |
| grid, origin axes | `View2D.SetGrid`, `SetAxesVisible` |
| export | `View2D.RenderImage`, `RenderSVG`; printing ([printing.md](printing.md)) |
| a part in 3D: extruded sketches, revolved profiles, meshes of files | `NewExtrudeMesh`, `NewLatheMesh`, `LoadOBJ`, `LoadSTL` ([3d.md](3d.md)) |
| edges of a drawing on a 3D view: creases and the outline of curved sides | `Material.EdgeColor` |
| section views | `Scene3D.ClipPlanes`, `SectionColor` |
| standard views, orthographic, axes gizmo, picking | `View3D` |

## A view of a drawing

The program keeps its shapes in a `SpatialIndex` and paints those the
index finds in the part of the drawing to paint, one path a layer:

```go
view.SetPainter(func(p *ui.View2DPaint) {
	for _, l := range layers {
		l.path.Reset()
	}
	index.Search(p.Visible, func(s *Shape) bool {
		if b := s.Bounds(); max(b.Width(), b.Height()) < 2*p.PixelSize() {
			s.Layer.dots.Rect(b.MinX, b.MinY, p.PixelSize(), p.PixelSize()) // too small to see: a dot
		} else {
			s.AddTo(s.Layer.path)
		}
		return true
	})
	for _, l := range layers {
		p.Canvas.StrokePath(l.path, ui.StrokeStyle{Width: l.Width, Dash: l.Dash, NonScaling: true}, l.Color)
	}
})
```

- One path a layer: a call to `StrokePath` costs much more than a line in it.
- Shapes smaller than a pixel or two drawn as dots: when all of a large
  drawing is in view, most shapes are such.
- Text smaller than a pixel and a half drawn as a box.
- What follows the pointer - the shape under it, the rubber band, the snap
  marker - goes in `SetOverlay`: the drawing is not painted again.
- After a change of the drawing, `view.Invalidate()`.

Picking and snapping use the index too: `index.Nearest(point, tolerance, n)`
with the tolerance `pickBox / view.Scale()`, then the real distance to each
candidate.

## Performance

On a 4-core desktop CPU of 2014 (Core i5-4460), the view of the example of
1060 × 700 pixels:

| | |
|---|---|
| the sample drawing (80 shapes, text, hatch, dimensions) | 4–7 ms |
| 100 000 shapes, all in view | about 100 ms |
| 100 000 shapes, zoomed in | 1–4 ms |
| a drag: the picture moved, the strip that comes in painted | a few ms |
| a move of the pointer: the overlay over the kept picture | under a millisecond plus the copy |
| the 3D part (1 400 triangles, edges, smoothed), 1050 × 700 | 40–50 ms; while turning, without smoothing, 3–4 times less |

The drawing is painted on the CPU. A drawing of millions of shapes all in
view takes seconds: such programs draw the far view from a simpler copy of
the drawing (a level of detail), which is theirs to make.

## What nui does not do

- The geometry kernel: offsets, fillets, trims, booleans of solids, NURBS,
  constraints - the program's.
- Files of CAD programs: DXF, DWG, STEP, IGES. nui reads OBJ and STL meshes
  and writes OBJ, PNG and SVG.
- A GPU: the views are painted by the CPU on all its cores. For models of
  millions of triangles at full frame rate a GPU renderer is needed.
- Undo, commands, the command line - the program's.
- Shadows, reflections and textures with mipmaps in 3D; caps of sections of
  open surfaces.
