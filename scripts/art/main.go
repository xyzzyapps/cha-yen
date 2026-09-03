package main
import ("fmt";"image";_ "image/png";"os";"strings")
func main(){f,_:=os.Open(os.Args[1]);img,_,_:=image.Decode(f);b:=img.Bounds()
W,H:=b.Dx(),b.Dy();CW,CH:=8,16
for by:=0; by<H; by+=CH{var row strings.Builder
 for bx:=0;bx<W;bx+=CW{sum,n:=0,0
  for y:=by;y<by+CH&&y<H;y++{for x:=bx;x<bx+CW&&x<W;x++{r,g,bl,_:=img.At(x,y).RGBA();sum+=int((r>>8+g>>8+bl>>8)/3);n++}}
  l:=float64(sum)/float64(n)
  switch{case l<25:row.WriteByte('.')
   case l<80:row.WriteByte(':')
   case l<160:row.WriteByte('+')
   case l<220:row.WriteByte('#')
   default:row.WriteByte('@')}}
 fmt.Println(row.String())}}
