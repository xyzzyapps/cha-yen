// Command kbprobe analyzes a screencap PNG and prints the topmost y where the
// bottom screen region becomes mostly light (the soft keyboard surface),
// plus the row of the first letter band. Usage: go run scripts/kbprobe.go s.png
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// Scan from bottom up: find first row (from top of lower half) that is
	// uniformly light (keyboard background).
	for y := h / 2; y < h; y++ {
		light, total := 0, 0
		for x := 0; x < w; x += 4 {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			l := (r>>8 + g>>8 + bl>>8) / 3
			total++
			if l > 200 {
				light++
			}
		}
		if light*100/total > 92 {
			fmt.Printf("keyboard_top=%d screen=%dx%d\n", y, w, h)
			// Gboard: letters start ~ 2 rows below top (after suggestions row)
			fmt.Printf("suggest_tap y=%d (q approx x=60)\n", y+(h-y)*3/10)
			return
		}
	}
	fmt.Println("no bright keyboard region found (dark theme keyboard?)")
}
