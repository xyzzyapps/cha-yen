// Command rowscan prints average luminance of every 40px row band of a PNG
// to reveal the screen layout (dark app area, toolbar, bright keyboard).
// Usage: go run .\scripts\rowscan t.png
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
)

func main() {
	f, _ := os.Open(os.Args[1])
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y += 40 {
		total, light, mid := 0, 0, 0
		for x := 0; x < w; x += 4 {
			if y+(0) >= b.Max.Y {
				break
			}
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			l := (r>>8 + g>>8 + bl>>8) / 3
			total++
			if l > 190 {
				light++
			} else if l > 55 && l < 130 {
				mid++
			}
		}
		fmt.Printf("y=%4d light=%3d%% mid=%3d%%\n", y, light*100/total, mid*100/total)
	}
}
