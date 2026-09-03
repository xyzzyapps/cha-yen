// Command tprobe finds the app toolbar buttons in a screencap PNG:
// scans a horizontal band for runs of "button grey" pixels (lighter than the
// dark app background) and prints their x-ranges.
// Usage: go run .\scripts\tprobe t.png <y>
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strconv"
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
	y, _ := strconv.Atoi(os.Args[2])
	b := img.Bounds()
	w := b.Dx()

	inRun := false
	var start int
	for x := 0; x < w; x++ {
		r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
		l := (r>>8 + g>>8 + bl>>8) / 3
		// button fill on fyne dark theme: mid grey (~60-110); bg ~30; text near white
		button := l > 55 && l < 130
		if button && !inRun {
			inRun, start = true, x
		} else if !button && inRun {
			inRun = false
			if x-start > 30 {
				fmt.Printf("button x=%d..%d center=%d\n", start, x, (start+x)/2)
			}
		}
	}
	if inRun {
		fmt.Printf("button x=%d..%d center=%d\n", w, w, w)
	}
}
