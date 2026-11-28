# Audit: remaining widgets, 3D/2D views, spatial index, fonts (nui/ui)

Scope: ui/canvas3d_*.go, view2d.go, view3d.go, zoomview.go, spatial.go, slider.go, scrollbarwidget.go,
progressbar.go, toggleswitch.go, checkbox.go, radiobutton.go, button*.go, tool_button.go, imagebox.go,
expander.go, groupbox.go, label.go, link.go, bars.go, accessible*.go, theme*.go, font_*.go, systemfont.go.

Reproducers (package `ui` internal tests) are saved in
`doc/audit/_repro/misc/`
(`zz_audit_misc_test.go`, `zz_audit_misc2_test.go`, `zz_audit_misc3_test.go`, `zombie_main.go`).
To run one: copy the files into `ui/`, then run `go test -run 'TestAuditXxx' ./ui/ -count=1 -v -timeout 60s`.
The test helpers `newTestForm` (controls_test.go) and `auditPaint` (in zz_audit_misc_test.go, which calls
`form.paintScaled` on a 500x400 RGBA) are used.

---

## 1. View2D paint hangs forever (float loop counter stuck) when the view is far from the origin

- Severity: **high** (the UI thread freezes, so the application hangs)
- File: ui/view2d.go:693, 694, 706, 717 (`paintGrid`)
- Root cause: the grid loops are `for i := i0; i <= i1; i++` with `i0, i1` of type **float64**
  (`math.Floor(r.MinX/step)` and similar). The guard `i1-i0 > 1e4` limits only the count of lines, not the
  magnitude. Once `|i0| >= 2^53`, `i++` rounds back to `i`, so the loop never ends. This happens when
  `|center| / step >= ~9e15`. Example: a drawing in millimetres of map coordinates (UTM northing 5e9 mm),
  zoomed in to the default `maxScale` of 1e9. It also happens with a fixed `Grid.Step` and a large
  coordinate. The Dots branch (line 693/694) has the same loop.
- Reproducer:
  ```go
  func TestAuditView2DGridFar(t *testing.T) {
      v := NewView2D()
      f := newTestForm(t, v)
      v.SetView(PointF{1e10, 0}, 1e9)
      done := make(chan struct{})
      go func() { defer close(done); auditPaint(t, f) }()
      select {
      case <-done:
      case <-time.After(10 * time.Second):
          t.Fatal("View2D paint hangs: grid loop at center 1e10, scale 1e9")
      }
  }
  // also: v.SetGrid(View2DGrid{Visible: true, Step: 1}); v.SetView(PointF{0, 3e16}, 10)
  ```
  Output: `View2D paint hangs: grid loop at center 1e10, scale 1e9` (both variants, 10 s timeout hit).
- Expected: the grid is drawn, or skipped when it cannot be resolved. Paint always ends.
- Fix direction: iterate with an integer counter `for k := 0; k <= n; k++ { i := i0 + float64(k) }` (n = int(i1-i0)).
  Also skip the grid when `math.Abs(i0) > 1<<52` or the step is below the ulp of the coordinate
  (`i*step` cannot be told apart then).

## 2. A 3D label at a non-finite or very far point panics the render (`slice bounds out of range`)

- Severity: **medium** (the paint of a View3D / `Canvas.Draw3D` crashes the app)
- File: ui/canvas3d_render.go:1243 (`drawLabels`). The panic is raised later, in text_glyph.go:316 `blendGlyphMask`.
- Root cause: `drawLabels` takes `lx, cy` from `cam.Project` and passes `int(math.Round(lx))-150` to
  `DrawText` without checking that they are finite and in range. For a NaN/Inf point, `int(NaN)` is
  `MinInt64`. For a far but finite point (1e300 off-axis), x is around 1e300 and the int conversion
  overflows. In both cases `DrawText` gets coordinates near ±2^63, and the glyph blitter's arithmetic
  overflows. Meshes, lines and points with the same NaN/Inf coordinates do **not** panic: they are
  filtered. Only labels are not.
- Reproducer:
  ```go
  sc := NewScene3D()
  sc.AddLabel(V3(math.NaN(), 0, 0), "x", color.RGBA{0, 0, 0, 255})   // or V3(1e300, 0, 0)
  NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(0, 0, 5), V3(0, 0, 0)))
  ```
  Output:
  ```
  panic: runtime error: slice bounds out of range [-59:]
  ui.blendGlyphMask      ui/text_glyph.go:316
  ui.drawTextGlyphs      ui/text_glyph.go:436
  ui.drawText            ui/text_renderer.go:210
  ui.(*Canvas).drawTextLines  ui/canvas.go:1456   (x=0x7fffffffffffff6a)
  ui.(*Canvas).DrawText
  ```
  At 1e300: `slice bounds out of range [:-9223372036854775749]`.
- Expected: a label whose projected point is not finite, or is far outside the view, is skipped.
- Fix direction: in `drawLabels`, `continue` unless `lx`, `ly` are finite and within, for example,
  `[-w-1000, 2w+1000]` (and the same band vertically). DrawText should also clamp or reject extreme
  coordinates (canvas area).

## 3. Font face cache grows without bound: a new face per distinct font size (zoomed text, NaN size)

- Severity: **medium** (memory leak in long-running apps with zoom or animation)
- File: ui/font_registry.go:136 (`getFaceEntry`, key `faceKey{name, size}` with `size float64`)
- Root cause: `faceCache` is keyed by the exact float64 size. It is never trimmed: it is only reset by
  RegisterFont/AddFallbackFont. `Canvas.drawTextXf` (canvas_transform.go:506) renders text under a
  transform at `fontSize * meanScale * scale`. A smooth zoom (View2D/ZoomView painter, an animated
  transform) therefore creates a new `opentype` face plus a fallbackFace for **every frame's** scale.
  A NaN size never matches itself as a map key, so each call with a NaN size adds an entry.
- Reproducer:
  ```go
  cnv, _ := NewImageCanvas(200, 100, 1)
  for i := 0; i < 2000; i++ {
      cnv.Save()
      k := 1 + float64(i)*0.0005          // a smooth zoom animation
      cnv.SetTransform(ScalingMatrix(k, k))
      cnv.SetFontSize(12)
      cnv.DrawText(0, 0, 100, 20, "Label")
      cnv.Restore()
  }
  // len(faceCache): 1 -> 2001
  // direct: 20000 MeasureText calls at distinct sizes + 1000 with NaN size:
  //   faceCache 1 -> 21000 entries, heap 0 -> 88 MB
  ```
- Expected: the memory stays bounded. Sizes are quantized, or the cache is an LRU.
- Fix direction: quantize the size in the key (e.g. to 1/64 px, which matches fixed.Int26_6), reject
  non-finite or non-positive sizes early (return an error or use a default), and cap the cache size
  (LRU, or clear when over N entries, as is already done for `fallbackFace` maps).

## 4. ToolButton.SetEnabled(false) is ignored while a parent is disabled

- Severity: **medium** (wrong state: a button that should stay disabled becomes enabled again)
- File: ui/tool_button.go:79
- Root cause: `if c.Enabled() == enabled { return }`. `Enabled()` takes the parents into account
  (widget.go:356). Under a disabled parent `Enabled()` is already false, so `SetEnabled(false)` returns
  without storing the widget's own flag. When the parent is enabled again, the button is enabled.
  `ToolBar.AddAction` syncs `b.SetEnabled(a.enabled)`. So an Action disabled while its toolbar (or the
  form's panel) is disabled, for example during a long operation, shows an enabled button afterwards.
  The disabled image is not refreshed either when only the parent changes.
- Reproducer:
  ```go
  panel := NewPanel()
  b := NewToolButton(image.NewRGBA(image.Rect(0, 0, 16, 16)), "tip", func() {})
  panel.AddWidget(0, 0, b)
  newTestForm(t, panel)
  panel.SetEnabled(false)
  b.SetEnabled(false)
  panel.SetEnabled(true)
  // b.Enabled() == true   <- wrong
  ```
  Output: `ToolButton.SetEnabled(false) under a disabled parent was ignored: enabled again with the parent`
- Expected: `b.Enabled()` is false.
- Fix direction: compare with the widget's own flag (`GetPropBool("enabled", true)`), or always call
  `ButtonImage.SetEnabled` and `showImage`. Choose the image from `Enabled()` at paint time, not at set time.

## 5. ZoomView: a NaN zoom (or a zoom range with 0) leaves the offset NaN for good

- Severity: **low-medium** (the view goes blank, and ActualSize/zoom keys don't recover it)
- File: ui/zoomview.go:132 (`ZoomAt`), 181-191 (`clampOffset`), 114 (`SetZoomRange`)
- Root cause: `math.Max(c.minZoom, math.Min(c.maxZoom, NaN))` is NaN, so `zoom` becomes NaN and
  `offX = cx - x/zoom` becomes NaN. Later, `ZoomAt(1)` computes `cx` from the NaN offset.
  `clampOffset` keeps NaN for a content larger than the view (`math.Max(0, math.Min(NaN, …))` is NaN).
  `SetZoomRange(0, …)` lets repeated zoom-out reach `zoom == 0`, so `x/0` gives Inf, then NaN, with
  the same permanent result. Only FitToView/CenterOn reset the offset.
- Reproducer:
  ```go
  z := NewZoomView(); newTestForm(t, z)
  z.SetContentSize(5000, 5000)
  z.SetZoom(math.NaN())
  z.ActualSize()
  cx, cy := z.ToContent(10, 10)   // NaN, NaN   (zoom reports 1)
  // and: z2.SetZoomRange(0, 64); 5000x z2.SetZoom(z2.Zoom()/1.25); z2.ActualSize() -> ToContent NaN, NaN
  ```
  Output: `after SetZoom(NaN)+ActualSize: zoom=1 content at (10,10)=NaN,NaN`, `minZoom 0: zoom=1 content at (10,10)=NaN,NaN`
- Expected: a NaN or non-positive zoom is ignored (View2D.clampScale already does this). The offset is always finite.
- Fix direction: in ZoomAt, return early if `!(zoom > 0)` or the zoom is not finite. In SetZoomRange, require `0 < min <= max`.
  In clampOffset, reset a non-finite offset.

## 6. Slider accepts NaN: value, range and step become NaN and the slider is stuck (also through accessibility)

- Severity: **low-medium**
- File: ui/slider.go:154 (`snap`), 134 (`SetValue`), 94 (`SetRange`), 110 (`SetStep`); ui/accessible.go:148 (`SetCurrent`)
- Root cause: `snap` uses `math.Max(min, math.Min(max, v))`, which returns NaN for a NaN `v`. The NaN
  is stored and `OnValueChanged` fires with NaN. After that every key gives NaN (`NaN + step`).
  `SetRange(NaN, x)` stores a NaN min. `SetStep(NaN)` stores NaN (`math.Max(0, NaN)` is NaN), and so
  does `SetStep(+Inf)` (`(v-min)/Inf*Inf` is NaN): `SetValue(5)` then gives NaN. An assistive
  technology calling `AccessibleNode.SetCurrent(NaN)` sets a NaN value (the builtin `min`/`max`
  propagate NaN). posOf/draw then convert NaN to int.
- Reproducer:
  ```go
  s := NewSlider(); f := newTestForm(t, s)
  s.SetRange(0, 10)
  s.SetValue(math.NaN())                  // Value() == NaN
  s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})  // still NaN
  s.SetStep(math.Inf(1)); s.SetValue(5)   // Value() == NaN
  // accessible: f.Accessible().Find(RoleSlider, "").SetCurrent(math.NaN()) -> Value() == NaN
  ```
  Output: `value after SetValue(NaN) = NaN`, `after right arrow = NaN`, `step inf: value NaN`,
  `after accessible SetCurrent(NaN): slider value NaN`
- Expected: NaN values are ignored. A NaN or Inf step and range bounds are rejected.
- Fix direction: in SetValue/changeValue, ignore non-finite values. In SetStep/SetRange, reject
  non-finite input. In `AccessibleNode.SetCurrent`, reject NaN.

## 7. Slider: a value at the maximum is not on the step grid when (max-min) is not a multiple of step

- Severity: **low** (wrong value)
- File: ui/slider.go:154-160 (`snap`)
- Root cause: `v = min + round((v-min)/step)*step` and then `v = math.Min(v, max)`. When rounding goes
  past `max`, the value is clamped to `max`, which is off the grid. SetStep is documented as "makes the
  value a multiple of step from the minimum".
- Reproducer: `SetRange(0,10); SetStep(4); SetValue(10)` gives 10. ArrowRight from 8 gives 10. End gives 10.
  Output: `value 10 is not on the step 4 from 0`
- Expected: 8, the largest multiple of the step that is <= max.
- Fix direction: when the rounded value is > max, use `min + math.Floor((max-min)/step)*step`.

## 8. Mesh with a negative index panics the renderer and ComputeNormals

- Severity: **low-medium** (a public struct field; a corrupt or computed index crashes the paint)
- File: ui/canvas3d_render.go:605 (`assembleMesh`), ui/canvas3d_mesh.go:100 (`ComputeNormals`),
  :218 (`wireEdges`), ui/canvas3d_scene.go:343 (`Pick`)
- Root cause: the index checks are only `ia >= n || ib >= n || ic >= n`. Indices are `[]int`, and a
  negative one goes straight to `r.clipV[ia]` / `m.Positions[a]`. The code is clearly meant to tolerate
  bad indices: it skips the out-of-range ones.
- Reproducer:
  ```go
  m := NewBoxMesh(1, 1, 1)
  m.Indices = append(m.Indices, -1, 0, 1)
  sc := NewScene3D(); sc.Add(m, Material{})
  NewRenderer3D().RenderImage(64, 64, sc, NewCamera(V3(3, 3, 3), V3(0, 0, 0)))
  // and (&Mesh{Positions: 3 pts, Indices: []int{0, 1, -2}}).ComputeNormals()
  ```
  Output: `panic: runtime error: index out of range [-1]`, and `index out of range [-2]` for ComputeNormals.
- Expected: triangles with negative indices are skipped, like the ones that are too large.
- Fix direction: use an unsigned check `uint(ia) >= uint(n)` in all four places (and in the
  `wireEdges`/`outlineEdges` users).

## 9. OpenURL leaves a zombie process per opened link (Linux/macOS)

- Severity: **low** (resource leak: one zombie per click on a Link, until the app exits)
- File: ui/link.go:167
- Root cause: `return cmd.Start()` without `cmd.Wait()`. xdg-open/open exit at once, and their entry
  stays in the process table as a zombie because Go does not reap children that are not waited for.
  (No shell is involved, so there is no shell command injection. A URL starting with `-` is passed as an
  option to xdg-open/open. A scheme allow-list may be worth having, but that is a design choice.)
- Reproducer (stand-alone, same call pattern), zombie_main.go:
  ```go
  for i := 0; i < 5; i++ { exec.Command("true").Start() }
  time.Sleep(500 * time.Millisecond)
  out, _ := exec.Command("sh", "-c", "ps -o pid,stat,comm --ppid $PPID").Output()
  ```
  Output: five lines `NNNN Z    true`.
- Expected: no zombies.
- Fix direction: `if err := cmd.Start(); err != nil { return err }; go cmd.Wait(); return nil`.

## 10. View3D: the zoom keys have no limits, so the distance becomes +Inf or 0 and the view breaks (target NaN)

- Severity: **low**
- File: ui/view3d.go:653, 655 (`keyDown`). The wheel at :623 only has a lower bound.
- Root cause: `v.distance *= 0.85` and `v.distance /= 0.85` are not clamped. After about 4400 key
  repeats of `-` the distance is +Inf. The camera position becomes Inf, and a Shift+arrow pan makes
  `target` NaN (`k := 2*Inf*tan/h`, then `Inf*0`). After that, every view is NaN until Home/R. With `+`
  the distance underflows to 1.5e-323: the camera sits at the target and the view is degenerate. A wheel
  delta with a large DY (`math.Pow(0.85, -5000)`) also gives +Inf at once.
- Reproducer: 5000x `v.keyDown(KeyMinus, ...)`, then `v.keyDown(KeyArrowLeft, {Shift: true})`.
  Output: `distance after 5000 minus: +Inf`, `after pan: target {NaN NaN NaN} distance +Inf`.
  6000x `KeyEqual`: `distance 1.5e-323`.
- Expected: the distance is clamped to a sane range (for example 1e-6 .. 1e3 x scene radius), as the wheel does for the lower end.
- Fix direction: route all zoom paths through one `setDistance` that clamps the value and ignores non-finite input.
  SetView should reject a NaN distance as well (`math.Max(NaN, 1e-9)` is NaN).

## 11. AccessibleRole.String() panics for a negative role

- Severity: **low**
- File: ui/accessible.go:73-74
- Root cause: `if int(r) < len(roleNames) { return roleNames[r] }` has no lower bound. AccessibleRole is
  a public `int` type, and `SetAccessibleRole` takes any value.
- Reproducer: `AccessibleRole(-1).String()` gives `panic: runtime error: index out of range [-1]`
- Fix direction: `if r >= 0 && int(r) < len(roleNames)`.

## 12. Text STL: a repeated `endfacet` adds the same triangle again; a facet without a normal inherits the previous one

- Severity: **low** (malformed-input handling: wrong mesh)
- File: ui/canvas3d_mesh.go:730 (`LoadSTL`, text branch)
- Root cause: `k` is reset only on `facet`. After `endfacet` with `k == 3` the triangle is added, but
  `k` stays 3, so each later `endfacet` adds it again. Vertices outside any facet are accepted too.
  `nrm` is not reset on `facet` with fewer than 5 fields, so it keeps the previous facet's normal (this
  only matters when that normal points the same side).
- Reproducer: `LoadSTL("solid\nvertex 1 2 3\nvertex 1 2 3\nvertex 1 2 3\nendfacet\nendfacet\n")` gives
  6 positions (two triangles) and no error.
- Fix direction: set `k = 0` after `endfacet`, set `nrm = Vec3{}` on each `facet`, and reject
  `vertex` outside facet/endfacet.

## 13. (verified by reading) macOS: registerSystemFontFile leaks an open file per skipped style

- Severity: **low** (macOS only; file descriptor leak per RegisterSystemFont/UseSystemFont call)
- File: ui/systemfont.go:178-182, ui/font_system.go:286-290 (`openFontFile`)
- Root cause: `openFontFile` returns `collection.Font(0)` with the `*os.File` left open ("stays open while
  the font is in use") when no font has the PostScript name (`exact == false`).
  `registerSystemFontFile` then drops that font for bold/italic (`if style != FontRegular && (err != nil || !exact) { continue }`)
  without closing the file. For a variable system font (SF), up to 3 descriptors leak per call. In
  `openFontFile`, the file also leaks when `collection.Font(0)` returns an error.
- Fix direction: return the file (or a closer) from `openFontFile` and close it when the font is not used.

---

### Checked, no bug found
- SpatialIndex: boxes at ±1.7e308, boxes spanning the whole float range, Remove after the root grew,
  NaN/Inf boxes (they are rejected): no hang or panic, and the search results are right.
- TriangulatePolygon/NewExtrudeMesh: 20 000 random small polygons with holes (degenerate, collinear,
  self-intersecting, repeated points) gave no panic, no out-of-range index and no NaN normal. Note:
  a collinear outline returns degenerate triangles instead of the nil the doc promises ("Nil if the
  polygon has no area"). This is cosmetic.
- Renderer: NaN/Inf/1e300 in mesh vertices, lines and points, NaN/huge line widths and point sizes, a
  camera at its target, Near == Far, a NaN Near, and orthographic with a zero distance: no panic.
- LoadOBJ: negative/huge/overflowing indices, missing vt/vn references, NaN coordinates: errors are
  reported correctly.
- RegisterFont fuzzing (mutated JetBrains Mono, about 6000 execs, 100 s) then MeasureText, renderText,
  Path.Text, DrawText: no crash.
- ProgressBar with a zero or inverted range: NaN or Inf percentages are clamped or not drawn, and there is no panic.
