package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

// render 将 SVG 渲染为指定尺寸的 RGBA 图像（透明背景）。直接按目标尺寸绘制，避免二次缩放产生黑边。
func render(r io.Reader, size int) image.Image {
	icon, err := oksvg.ReadIconStream(r)
	if err != nil {
		panic(err)
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1)
	return rgba
}

// encodeMultiICO 将多张位图写入标准多尺寸 .ico 文件（32bpp BGRA + alpha，AND 掩码全 0）。
func encodeMultiICO(path string, imgs []image.Image) error {
	var datas [][]byte
	entryBuf := &bytes.Buffer{}
	pos := 6 + len(imgs)*16
	for _, im := range imgs {
		b := im.Bounds()
		w, h := b.Dx(), b.Dy()
		bw := &bytes.Buffer{}
		bi := make([]byte, 40)
		binary.LittleEndian.PutUint32(bi[0:4], 40)
		binary.LittleEndian.PutUint32(bi[4:8], uint32(w))
		binary.LittleEndian.PutUint32(bi[8:12], uint32(h*2)) // XOR + AND 总高度
		binary.LittleEndian.PutUint16(bi[12:14], 1)          // planes
		binary.LittleEndian.PutUint16(bi[14:16], 32)         // bpp
		bw.Write(bi)
		// XOR：自下而上、BGRA
		for y := h - 1; y >= 0; y-- {
			for x := 0; x < w; x++ {
				c := im.At(b.Min.X+x, b.Min.Y+y).(color.RGBA)
				bw.WriteByte(c.B)
				bw.WriteByte(c.G)
				bw.WriteByte(c.R)
				bw.WriteByte(c.A)
			}
		}
		// AND 掩码：全 0（透明由 alpha 通道表达）
		andRow := ((w + 31) / 32) * 4
		bw.Write(make([]byte, andRow*h))
		data := bw.Bytes()
		datas = append(datas, data)

		w8, h8 := byte(w), byte(h)
		if w == 256 {
			w8, h8 = 0, 0
		}
		entryBuf.Write([]byte{w8, h8, 0, 0}) // width,height,colorCount,padding
		entryBuf.Write([]byte{1, 0})         // planes
		entryBuf.Write([]byte{32, 0})        // bpp
		binary.Write(entryBuf, binary.LittleEndian, uint32(len(data)))
		binary.Write(entryBuf, binary.LittleEndian, uint32(pos))
		pos += len(data)
	}
	out := &bytes.Buffer{}
	out.Write([]byte{0, 0, 1, 0}) // reserved, type=1
	out.Write([]byte{byte(len(imgs)), 0})
	out.Write(entryBuf.Bytes())
	for _, d := range datas {
		out.Write(d)
	}
	return os.WriteFile(path, out.Bytes(), 0644)
}

// opaqueCount 统计不透明像素数。
func opaqueCount(img image.Image) int {
	b := img.Bounds()
	o := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				o++
			}
		}
	}
	return o
}

func main() {
	svg := "appicon.svg"
	outPath := "appicon.ico"
	if len(os.Args) > 1 {
		svg = os.Args[1]
	}
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}

	sizes := []int{256, 128, 64, 48, 32, 16}
	imgs := make([]image.Image, 0, len(sizes))
	for _, s := range sizes {
		f, err := os.Open(svg)
		if err != nil {
			panic(err)
		}
		imgs = append(imgs, render(f, s))
		f.Close()
	}
	for i, s := range sizes {
		fmt.Printf("size %d opaque: %d\n", s, opaqueCount(imgs[i]))
	}

	if err := encodeMultiICO(outPath, imgs); err != nil {
		panic(err)
	}

	// 导出 256 预览 PNG 便于人工核对
	prev, _ := os.Create(strings.TrimSuffix(outPath, ".ico") + "_preview.png")
	png.Encode(prev, imgs[0])
	prev.Close()

	fmt.Println("generated", outPath, "with", len(imgs), "sizes")
}
