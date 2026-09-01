package lfm2boundary

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreflightRequiresExactLoadedVisionF16Context(t *testing.T){srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/api/v1/models"{t.Fatalf("path=%s",r.URL.Path)};w.Header().Set("Content-Type","application/json");_,_=w.Write([]byte(`{"models":[{"type":"llm","key":"lfm2-vl-1.6b","quantization":{"name":"F16"},"capabilities":{"vision":true},"loaded_instances":[{"id":"lfm2-vl-1.6b","config":{"context_length":4096,"parallel":4}}]}]}`))}));defer srv.Close();got,err:=Preflight(context.Background(),srv.URL+"/v1",RequiredModel);if err!=nil{t.Fatal(err)};if!got.Vision||got.ContextLength!=4096||got.ServerParallel!=4{t.Fatalf("got=%+v",got)}}
func TestPreflightRejectsWrongIdentityVisionOrContext(t *testing.T){if _,err:=Preflight(context.Background(),"http://127.0.0.1:1/v1","other");err==nil{t.Fatal("expected exact model rejection")};cases:=[]string{`{"models":[{"type":"llm","key":"lfm2-vl-1.6b","quantization":{"name":"F16"},"capabilities":{"vision":false},"loaded_instances":[{"id":"lfm2-vl-1.6b","config":{"context_length":4096}}]}]}`,`{"models":[{"type":"llm","key":"lfm2-vl-1.6b","quantization":{"name":"F16"},"capabilities":{"vision":true},"loaded_instances":[{"id":"lfm2-vl-1.6b","config":{"context_length":8192}}]}]}`};for _,body:=range cases{srv:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte(body))}));_,err:=Preflight(context.Background(),srv.URL,RequiredModel);srv.Close();if err==nil{t.Fatalf("expected rejection for %s",body)}}}
func TestDeclaredCropUsesNearestNeighborAndExcludesExactPlane(t *testing.T){img:=image.NewGray(image.Rect(0,0,640,640));for y:=0;y<640;y++{for x:=0;x<640;x++{img.SetGray(x,y,color.Gray{Y:uint8((x+y)%256)})}};var raw bytes.Buffer;if err:=png.Encode(&raw,img);err!=nil{t.Fatal(err)};specs:=DeclaredCrops();if len(specs)!=4{t.Fatalf("crops=%d",len(specs))};for _,s:=range specs{if s.ID=="SEMANTIC_FULL"&&s.Y1>=398{t.Fatalf("semantic crop reaches exact plane: %+v",s)}};crop,err:=CropNearestPNG(raw.Bytes(),specs[0]);if err!=nil{t.Fatal(err)};cfg,err:=png.DecodeConfig(bytes.NewReader(crop));if err!=nil{t.Fatal(err)};if cfg.Width!=(632-8)*3||cfg.Height!=(105-8)*3{t.Fatalf("size=%dx%d",cfg.Width,cfg.Height)};if strings.TrimSpace(specs[0].ID)==""{t.Fatal("missing id")}}
