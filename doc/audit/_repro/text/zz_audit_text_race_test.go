package ui

import (
	"fmt"
	"image"
	"sync"
	"testing"
)

func TestAuditTextRace(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 6; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			cnv := NewCanvasScaled(image.NewRGBA(image.Rect(0, 0, 200, 50)), 1)
			for i := 0; i < 40; i++ {
				s := fmt.Sprintf("x%d שלום 😀 é %d", i, g)
				MeasureText(FontFamilySans, 12+float64(i%3), s)
				GetCharPositions(FontFamilySans, 12, s)
				TextCaretX(FontFamilySans, 12, s, i%5)
				cnv.DrawText(0, 0, 200, 50, s)
				cnv.DrawTextF(1, 1, s)
				l := NewTextLayout(s, TextStyle{Size: 14, Bold: i%2 == 0})
				cnv.DrawTextLayout(l, 2, 2)
				p := NewPath()
				p.Text(0, 0, s, FontFamilySans, 12)
				cnv.DrawTextRotated(10, 10, 0.3, s)
			}
		}(g)
	}
	wg.Wait()
}
