package main
import ("fmt";"image";_ "image/png";"os")
func main(){f,_:=os.Open(os.Args[1]);img,_,_:=image.Decode(f);b:=img.Bounds()
w:=b.Dx()
for y:=100;y<600;y+=24{light:=0
 for x:=0;x<w;x+=3{r,g,bl,_:=img.At(x,y).RGBA();if (r>>8+g>>8+bl>>8)/3>150{light++}}
 bar:=""
 for i:=0;i<light/8;i++{bar+="#"}
 fmt.Printf("y=%3d %s\n",y,bar)}}
