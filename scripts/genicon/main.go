package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// genicon writes a simple 512x512 "bubble tea" placeholder icon.
func main() {
	const s = 512
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	bg := color.RGBA{24, 26, 32, 255}     // terminal dark
	foam := color.RGBA{233, 226, 208, 255} // milk foam
	boba := color.RGBA{40, 34, 30, 255}
	tea := color.RGBA{196, 148, 92, 255}
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			c := bg
			switch {
			case y > 120 && y < 170 && x > 120 && x < 392:
				c = foam // foam band
			case y >= 170 && y < 420 && x > 130 && x < 382:
				c = tea // cup body
				dx, dy := float64(x-190), float64(y-350)
				if dx*dx+dy*dy < 900 {
					c = boba // pearls
				}
				dx2, dy2 := float64(x-300), float64(y-320)
				if dx2*dx2+dy2*dy2 < 900 {
					c = boba
				}
			}
			img.Set(x, y, c)
		}
	}
	f, _ := os.Create("assets/icon.png")
	defer f.Close()
	_ = png.Encode(f, img)
}
