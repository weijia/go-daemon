package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"testing"
)

func TestMakeIcon(t *testing.T) {
	ico := makeIcon()
	if len(ico) < 6+16 {
		t.Fatal("ICO 数据过小")
	}
	if ico[0] != 0 || ico[1] != 0 {
		t.Fatal("保留字错误")
	}
	if ico[2] != 1 || ico[3] != 0 {
		t.Fatal("不是图标类型")
	}
	if ico[4] != 1 || ico[5] != 0 {
		t.Fatal("图标数量错误")
	}
	// PNG-in-ICO：数据应以内置 PNG 签名开头
	if !bytes.HasPrefix(ico[22:], []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("ICO 未包含 PNG 数据")
	}
}

func TestPngToICOLayout(t *testing.T) {
	var pngBuf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	out := pngToICO(pngBuf.Bytes())
	// ICONDIRENTRY 中 PNG 数据偏移量字段（字节 18..22）应为 22
	off := binary.LittleEndian.Uint32(out[18:22])
	if off != 22 {
		t.Fatalf("期望偏移 22，得到 %d", off)
	}
	// 末尾 PNG 数据长度应与原始一致
	if len(out) != 22+pngBuf.Len() {
		t.Fatalf("长度不匹配: %d != %d", len(out), 22+pngBuf.Len())
	}
}
