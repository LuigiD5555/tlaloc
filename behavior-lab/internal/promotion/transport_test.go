package promotion

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func testPNG(t *testing.T)[]byte{t.Helper();img:=image.NewGray(image.Rect(0,0,640,640));for i:=range img.Pix{img.Pix[i]=0xff};for y:=100;y<540;y++{for x:=100;x<540;x++{if (x+y)%17==0{img.SetGray(x,y,color.Gray{Y:0})}}};var b bytes.Buffer;if err:=png.Encode(&b,img);err!=nil{t.Fatal(err)};return b.Bytes()}

func TestBuildTransportVariants(t *testing.T){
	variants,err:=BuildTransportVariants(testPNG(t));if err!=nil{t.Fatal(err)}
	if len(variants)!=4{t.Fatalf("expected 4 variants, got %d",len(variants))}
	want:=map[string][2]int{"original":{640,640},"resize-75":{480,480},"resize-50":{320,320},"jpeg-preview":{480,480}}
	for _,v:=range variants{size,ok:=want[v.Name];if !ok{t.Fatalf("unexpected variant %s",v.Name)};if v.Width!=size[0]||v.Height!=size[1]||len(v.Bytes)==0{t.Fatalf("bad variant %+v",v)};if v.Name=="jpeg-preview"&&v.MediaType!="image/jpeg"{t.Fatal("preview must be JPEG")}}
}
