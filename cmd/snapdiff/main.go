// snapdiff compares two folders of screenshots taken by TestSnapshots of
// the examples (see doc/testing.md): it lists the pictures that differ and
// the regions of the pixels that differ in each, and with -out writes each
// region before and after, side by side and zoomed in. With -tolerance the
// pixels whose components differ by that much at most count as the same:
// a change of the rounding of the blending moves many by one or two.
//
//	NUI_SNAPSHOT_DIR=/tmp/before go test ./examples -run TestSnapshots
//	... the change ...
//	NUI_SNAPSHOT_DIR=/tmp/after go test ./examples -run TestSnapshots
//	go run ./cmd/snapdiff -out /tmp/diff /tmp/before /tmp/after
//
// The exit status is 1 if any picture differs or is missing.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	out := flag.String("out", "", "folder for the zoomed pictures of the regions that differ")
	maxRegions := flag.Int("regions", 10, "regions listed per picture")
	tolerance := flag.Int("tolerance", 0, "the largest difference of a component that counts as the same")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: snapdiff [-out folder] [-regions n] [-tolerance n] before-folder after-folder")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	before, after := flag.Arg(0), flag.Arg(1)
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			fail(err)
		}
	}

	names, err := pngNames(before)
	if err != nil {
		fail(err)
	}
	afterNames, err := pngNames(after)
	if err != nil {
		fail(err)
	}
	inBefore := map[string]bool{}
	for _, n := range names {
		inBefore[n] = true
	}

	same, differ := 0, 0
	for _, name := range afterNames {
		if !inBefore[name] {
			fmt.Printf("%s: only after\n", name)
			differ++
		}
	}
	for _, name := range names {
		a, err := readPNG(filepath.Join(before, name))
		if err != nil {
			fail(err)
		}
		b, err := readPNG(filepath.Join(after, name))
		if os.IsNotExist(err) {
			fmt.Printf("%s: only before\n", name)
			differ++
			continue
		} else if err != nil {
			fail(err)
		}
		if a.Rect != b.Rect {
			fmt.Printf("%s: size %v -> %v\n", name, a.Rect.Size(), b.Rect.Size())
			differ++
			continue
		}
		regions := diffRegions(a, b, *tolerance)
		if len(regions) == 0 {
			same++
			continue
		}
		differ++
		total := 0
		for _, r := range regions {
			total += r.pixels
		}
		fmt.Printf("%s: %d pixels in %d regions\n", name, total, len(regions))
		for i, r := range regions {
			if i == *maxRegions {
				fmt.Printf("    ... %d more\n", len(regions)-i)
				break
			}
			fmt.Printf("    %6d px in %v, %dx%d, by up to %d\n", r.pixels, r.bounds, r.bounds.Dx(), r.bounds.Dy(), r.maxDelta)
			if *out != "" {
				file := fmt.Sprintf("%s-%d.png", strings.TrimSuffix(name, ".png"), i+1)
				if err := writePNG(filepath.Join(*out, file), sideBySide(a, b, r.bounds)); err != nil {
					fail(err)
				}
			}
		}
	}
	fmt.Printf("%d same, %d differ\n", same, differ)
	if differ > 0 {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "snapdiff:", err)
	os.Exit(2)
}

func pngNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func readPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Rect, img, img.Bounds().Min, draw.Src)
	return rgba, nil
}

func writePNG(path string, img image.Image) error {
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

// region is a group of differing pixels, each at most regionGap pixels
// from another of the group
type region struct {
	bounds image.Rectangle
	pixels int
	// The largest difference of a component of a pixel
	maxDelta int
}

const regionGap = 3

// diffRegions returns the regions of the pixels whose components differ by
// more than tolerance, the largest first
func diffRegions(a, b *image.RGBA, tolerance int) []region {
	w, h := a.Rect.Dx(), a.Rect.Dy()
	differs := make([]bool, w*h)
	deltas := make([]uint8, w*h)
	any := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := a.PixOffset(x, y)
			delta := 0
			for k := 0; k < 4; k++ {
				delta = max(delta, abs(int(a.Pix[i+k])-int(b.Pix[i+k])))
			}
			if delta > tolerance {
				differs[y*w+x] = true
				deltas[y*w+x] = uint8(delta)
				any = true
			}
		}
	}
	if !any {
		return nil
	}
	var regions []region
	seen := make([]bool, w*h)
	for start := range differs {
		if !differs[start] || seen[start] {
			continue
		}
		r := region{bounds: image.Rect(start%w, start/w, start%w+1, start/w+1)}
		stack := []int{start}
		seen[start] = true
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			px, py := p%w, p/w
			r.pixels++
			r.maxDelta = max(r.maxDelta, int(deltas[p]))
			r.bounds = r.bounds.Union(image.Rect(px, py, px+1, py+1))
			for dy := -regionGap; dy <= regionGap; dy++ {
				for dx := -regionGap; dx <= regionGap; dx++ {
					x, y := px+dx, py+dy
					if x < 0 || y < 0 || x >= w || y >= h {
						continue
					}
					if q := y*w + x; differs[q] && !seen[q] {
						seen[q] = true
						stack = append(stack, q)
					}
				}
			}
		}
		regions = append(regions, r)
	}
	sort.Slice(regions, func(i, j int) bool { return regions[i].pixels > regions[j].pixels })
	return regions
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// sideBySide returns the region with a margin around it, before on the
// left and after on the right, zoomed in so that a pixel is seen
func sideBySide(a, b *image.RGBA, r image.Rectangle) *image.RGBA {
	const margin = 12
	crop := r.Inset(-margin).Intersect(a.Rect)
	zoom := max(1, min(8, 600/max(crop.Dx(), crop.Dy())))
	gap := 4
	cw, ch := crop.Dx()*zoom, crop.Dy()*zoom
	res := image.NewRGBA(image.Rect(0, 0, cw*2+gap, ch))
	draw.Draw(res, res.Rect, image.NewUniform(color.RGBA{255, 0, 255, 255}), image.Point{}, draw.Src)
	for i, img := range []*image.RGBA{a, b} {
		left := i * (cw + gap)
		for y := 0; y < crop.Dy(); y++ {
			for x := 0; x < crop.Dx(); x++ {
				c := img.RGBAAt(crop.Min.X+x, crop.Min.Y+y)
				cell := image.Rect(left+x*zoom, y*zoom, left+(x+1)*zoom, (y+1)*zoom)
				draw.Draw(res, cell, image.NewUniform(c), image.Point{}, draw.Src)
			}
		}
	}
	return res
}
