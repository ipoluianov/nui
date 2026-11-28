//go:build linux

package platforms

import (
	"encoding/binary"
	"math"
	"net/url"
	"testing"
)

// The uri-list of a drag out (dragout_linux.go) read back by the receiving
// side (parseURIList): the path must survive
func TestAuditDragOutURIRoundTrip(t *testing.T) {
	for _, f := range []string{"/tmp/a b#c?d%e.txt", "docs/report.txt", "report.txt"} {
		uri := (&url.URL{Scheme: "file", Path: f}).String()
		got := parseURIList(uri + "\r\n")
		t.Logf("%q -> %q -> %q", f, uri, got)
		if len(got) != 1 || got[0] != f {
			t.Errorf("path %q sent as %q, received as %q", f, uri, got)
		}
	}
}

func TestAuditHeadlessScaleNaN(t *testing.T) {
	for _, s := range []float64{math.NaN(), math.Inf(1)} {
		w := &HeadlessWindow{width: 10, height: 10, scale: max(s, 1)}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("scale %v (as SetHeadlessScreen/MoveToScreen keep it): Render panics: %v", s, r)
				}
			}()
			w.Render()
		}()
	}
}

func FuzzAuditImageFromDIB(f *testing.F) {
	h := make([]byte, 40+16)
	le := binary.LittleEndian
	le.PutUint32(h[0:], 40)
	le.PutUint32(h[4:], 2)
	le.PutUint32(h[8:], 2)
	le.PutUint16(h[14:], 32)
	f.Add(h)
	h2 := append([]byte(nil), h...)
	le.PutUint32(h2[16:], 3)
	f.Add(h2)
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) >= 12 {
			// keep allocations small
			w := int32(binary.LittleEndian.Uint32(b[4:]))
			hh := int32(binary.LittleEndian.Uint32(b[8:]))
			if w > 2048 || hh > 2048 || hh < -2048 {
				return
			}
		}
		imageFromDIB(b)
	})
}
