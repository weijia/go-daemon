package main

import (
	"image/color"
	"testing"
)

// TestIconRegions 校验可爱图标各关键区域颜色符合预期（不依赖人工目检）。
func TestIconRegions(t *testing.T) {
	const S = 256.0
	// 四角应为透明
	for _, p := range [][2]float64{{4, 4}, {S - 4, 4}, {4, S - 4}, {S - 4, S - 4}} {
		r, g, b, a := iconPixel(p[0], p[1], S)
		if a != 0 {
			t.Fatalf("期望四角透明，但 (%v,%v) alpha=%d", p[0], p[1], a)
		}
		_ = r; _ = g; _ = b
	}
	// 主体中心应为蓝色且不透明
	cr, cg, cb, ca := iconPixel(S/2, S/2, S)
	if ca == 0 || !(cb > cg && cg > cr) {
		t.Fatalf("主体中心应为蓝色不透明，实际 (r=%d,g=%d,b=%d,a=%d)", cr, cg, cb, ca)
	}
	// 左眼（白底）区域：取眼睛上沿、避开瞳孔
	er, eg, eb, ea := iconPixel(S/2-S*0.17, S/2-S*0.12-S*0.10, S)
	if ea == 0 || er < 200 || eg < 200 || eb < 200 {
		t.Fatalf("眼睛应为白色，实际 (r=%d,g=%d,b=%d,a=%d)", er, eg, eb, ea)
	}
	// 嘴巴（深色）区域：下半圆中心
	mr, mg, mb, ma := iconPixel(S/2, S/2+S*0.2+S*0.05, S)
	if ma == 0 || mr > 80 || mg > 80 || mb > 80 {
		t.Fatalf("嘴巴应为深色，实际 (r=%d,g=%d,b=%d,a=%d)", mr, mg, mb, ma)
	}
	// 在线点（绿色）区域：右下角（圆内）
	gr, gg, gb, ga := iconPixel(S/2+S*0.24, S/2+S*0.24, S)
	if ga == 0 || gg <= gr || gg <= gb {
		t.Fatalf("在线点应为绿色，实际 (r=%d,g=%d,b=%d,a=%d)", gr, gg, gb, ga)
	}
	_ = color.RGBA{}
}
