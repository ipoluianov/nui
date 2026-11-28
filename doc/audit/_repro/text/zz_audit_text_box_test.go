package ui

import (
	"image/color"
	"runtime/debug"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAuditTextBoxFuzz(t *testing.T) {
	pieces := []string{"a", "b ", " ", "\n", "\t", "שלום", "مرحبا", "123", "😀", "👨‍👩‍👧", "🇩🇪", "é", "́", "‍", "‮", "क्षि", "\xff", "\x00", "ﷺ", "x̀́"}
	seed := uint32(7)
	rnd := func(n int) int { seed = seed*1664525 + 1013904223; return int(seed>>8) % n }
	str := func() string {
		var b strings.Builder
		for i := rnd(8); i > 0; i-- {
			b.WriteString(pieces[rnd(len(pieces))])
		}
		return b.String()
	}
	keys := []Key{KeyArrowLeft, KeyArrowRight, KeyArrowUp, KeyArrowDown, KeyHome, KeyEnd, KeyBackspace, KeyDelete, KeyEnter, KeyA, KeyZ, KeyY, KeyX, KeyC, KeyV}
	for run := 0; run < 4000; run++ {
		box := NewTextBox()
		form := newTestForm(t, box)
		box.Focus()
		var ops []string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic run %d: %v\nops: %s\n%s", run, r, strings.Join(ops, "; "), debug.Stack())
				}
			}()
			for step := 0; step < 80; step++ {
				switch rnd(16) {
				case 0:
					s := str()
					ops = append(ops, "SetText "+quote(s))
					box.SetText(s)
				case 1:
					s := str()
					ops = append(ops, "InsertText "+quote(s))
					box.InsertText(s)
				case 2, 3, 4:
					k := keys[rnd(len(keys))]
					m := KeyModifiers{Shift: rnd(3) == 0, Ctrl: rnd(4) == 0}
					ops = append(ops, "key "+quote(string(rune(k)))+" "+quote(m.String()))
					typeKey(form, k, 0, m)
				case 5, 6:
					s := pieces[rnd(len(pieces))]
					ops = append(ops, "type "+quote(s))
					for _, r := range s {
						form.processChar(r)
					}
				case 7:
					m := rnd(2) == 0
					ops = append(ops, "multiline "+quote(boolS(m)))
					box.SetMultiline(m)
				case 8:
					p := rnd(4) == 0
					ops = append(ops, "password "+quote(boolS(p)))
					box.SetIsPassword(p)
				case 9:
					a, c := TextPos{rnd(5) - 1, rnd(12) - 2}, TextPos{rnd(5) - 1, rnd(12) - 2}
					ops = append(ops, "SetSelection")
					box.SetSelection(a, c)
				case 10:
					x, y := rnd(300)-20, rnd(120)-20
					ops = append(ops, "mouse")
					box.MouseDown(MouseButtonLeft, x, y, KeyModifiers{})
					box.MouseMove(rnd(300)-20, rnd(120)-20, KeyModifiers{})
					box.MouseUp(MouseButtonLeft, x, y, KeyModifiers{})
				case 11:
					ops = append(ops, "align/visual")
					box.SetTextAlign(TextAlign(rnd(4)))
					box.SetVisualCursorMovement(rnd(2) == 0)
					box.SetLineNumbers(rnd(2) == 0)
				case 12:
					ops = append(ops, "highlighter")
					box.SetHighlighter(func(line int, text string) []TextSpan {
						return []TextSpan{{Start: rnd(6) - 2, End: rnd(10), Color: color.RGBA{255, 0, 0, 255}, Bold: rnd(2) == 0, Background: color.RGBA{0, 0, 255, 255}}, {Start: 1, End: 100, Italic: true}}
					})
				case 13:
					ops = append(ops, "SelectRange")
					box.SelectRange(rnd(10)-2, rnd(10)-2)
				case 14:
					o := rnd(40) - 5
					ops = append(ops, "PosAt")
					p := box.PosAt(o)
					if o >= 0 && o <= utf8.RuneCountInString(box.Text()) && box.OffsetOf(p) != o {
						t.Errorf("OffsetOf(PosAt(%d)) = %d, text %q", o, box.OffsetOf(p), box.Text())
					}
				case 15:
					ops = append(ops, "paint")
					paintForm(form)
				}
				// invariants
				if strings.Join(box.Lines(), "\n") != box.Text() {
					t.Fatalf("Text() and Lines() differ: %q vs %q ops %v", box.Text(), box.Lines(), ops)
				}
				cp := box.CursorPosition()
				if cp.Line < 0 || cp.Line >= box.LineCount() || cp.Col < 0 || cp.Col > utf8.RuneCountInString(box.Line(cp.Line)) {
					t.Fatalf("cursor %+v out of the text %q; ops %s", cp, box.Text(), strings.Join(ops, "; "))
				}
				from, to := box.Selection()
				for _, p := range []TextPos{from, to} {
					if p.Line < 0 || p.Line >= box.LineCount() || p.Col < 0 || p.Col > utf8.RuneCountInString(box.Line(p.Line)) {
						t.Fatalf("selection %+v out of text %q; ops %s", p, box.Text(), strings.Join(ops, "; "))
					}
				}
				_ = box.SelectedText()
			}
			paintForm(form)
		}()
	}
}

func quote(s string) string { return "`" + strings.ReplaceAll(s, "`", "'") + "`" }
func boolS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
