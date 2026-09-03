package main
import ("fmt";"image";_ "image/png";"os")
func main(){f,_:=os.Open(os.Args[1]);img,_,_:=image.Decode(f);b:=img.Bounds();d:=0;t:=0
for y:=0;y<b.Dy();y+=2{for x:=0;x<b.Dx();x+=2{r,g,bl,_:=img.At(b.Min.X+x,b.Min.Y+y).RGBA();l:=(r>>8+g>>8+bl>>8)/3;if l<80{d++};t++}}
fmt.Printf("dark=%d/%d (%.1f%%)\n",d,t,float64(d)*100/float64(t))}
