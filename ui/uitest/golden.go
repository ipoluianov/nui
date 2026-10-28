package uitest

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The golden images are in testdata/golden of the package under test, the
// ones that differ by the platform in testdata/golden/<GOOS> (see
// MatchGoldenImage).
// Write them anew with
//
//	go test ./mypackage -update-golden
//
// or with NUI_UPDATE_GOLDEN=1 in the environment, and look at them before
// committing: they are what the test expects from now on.
var updateGolden = flag.Bool("update-golden", false, "write the golden images of uitest anew")

// GoldenTolerance is how far a channel of a pixel may be from the golden
// image (0..255) without counting as a difference: the antialiasing of the
// text differs a little between machines
var GoldenTolerance = 8

// GoldenMaxDiff is the share of the pixels (0..1) that may differ from the
// golden image beyond GoldenTolerance
var GoldenMaxDiff = 0.0

// MatchGolden compares the screenshot of the window with the golden image
// testdata/golden/<name>.png. On a difference the test fails, and the
// screenshot and a picture of the differences (in red) are written next to
// the golden one, as <name>.actual.png and <name>.diff.png. A missing golden
// image is written and fails the test, so it is looked at before it counts.
func (w *Window) MatchGolden(name string) {
	w.t.Helper()
	MatchGoldenImage(w.t, name, w.Screenshot())
}

// MatchGoldenImage compares the image with the golden image of the name, as
// Window.MatchGolden
func MatchGoldenImage(t testing.TB, name string, img *image.RGBA) {
	t.Helper()
	dir := filepath.Join("testdata", "golden")
	// A picture that differs by the platform (the order of the dialog
	// buttons, the names of the shortcuts) has its own golden image in
	// testdata/golden/<GOOS>
	if osDir := filepath.Join(dir, runtime.GOOS); fileExists(filepath.Join(osDir, name+".png")) {
		dir = osDir
	}
	path := filepath.Join(dir, name+".png")
	actualPath := filepath.Join(dir, name+".actual.png")
	diffPath := filepath.Join(dir, name+".diff.png")

	if *updateGolden || os.Getenv("NUI_UPDATE_GOLDEN") != "" {
		if err := writePNG(path, img); err != nil {
			t.Fatalf("uitest: %v", err)
		}
		os.Remove(actualPath)
		os.Remove(diffPath)
		t.Logf("uitest: wrote %s", path)
		return
	}

	golden, err := readPNG(path)
	if os.IsNotExist(err) {
		if err := writePNG(path, img); err != nil {
			t.Fatalf("uitest: %v", err)
		}
		t.Errorf("uitest: no golden image %s: written, look at it and run the test again", path)
		return
	}
	if err != nil {
		t.Fatalf("uitest: %v", err)
	}

	diff, count := compareImages(golden, img)
	total := img.Bounds().Dx() * img.Bounds().Dy()
	if count == 0 || (golden.Bounds().Size() == img.Bounds().Size() && float64(count) <= GoldenMaxDiff*float64(total)) {
		os.Remove(actualPath)
		os.Remove(diffPath)
		return
	}
	writePNG(actualPath, img)
	writePNG(diffPath, diff)
	if golden.Bounds().Size() != img.Bounds().Size() {
		t.Errorf("uitest: %s: size %v, golden %v; see %s", name, img.Bounds().Size(), golden.Bounds().Size(), actualPath)
		return
	}
	t.Errorf("uitest: %s: %d of %d pixels differ; see %s and %s", name, count, total, actualPath, diffPath)
}

// compareImages returns a picture of the differences (the image dimmed, the
// differing pixels red) and how many pixels differ
func compareImages(golden image.Image, img *image.RGBA) (*image.RGBA, int) {
	b := img.Bounds()
	diff := image.NewRGBA(b)
	count := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := img.RGBAAt(x, y)
			var g color.RGBA
			inGolden := image.Pt(x, y).In(golden.Bounds())
			if inGolden {
				g = color.RGBAModel.Convert(golden.At(x, y)).(color.RGBA)
			}
			if !inGolden || far(a.R, g.R) || far(a.G, g.G) || far(a.B, g.B) || far(a.A, g.A) {
				count++
				diff.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
				continue
			}
			gray := uint8((uint16(a.R) + uint16(a.G) + uint16(a.B)) / 3 / 3)
			diff.SetRGBA(x, y, color.RGBA{gray, gray, gray, 255})
		}
	}
	return diff, count
}

func far(a, b uint8) bool {
	d := int(a) - int(b)
	return d > GoldenTolerance || -d > GoldenTolerance
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
