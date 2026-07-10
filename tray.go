package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

// runTray 启动系统托盘（阻塞直到退出）。
func runTray(app *App) {
	systray.Run(func() {
		onReady(app)
	}, func() {
		app.Quit()
	})
}

func onReady(app *App) {
	systray.SetIcon(makeIcon())
	systray.SetTitle("UFS Node")
	systray.SetTooltip("UFS Node - RemoteStorage 上报代理")

	mOpen := systray.AddMenuItem("打开配置", "在浏览器中打开配置页")
	mUpdate := systray.AddMenuItem("立即上报", "立即向服务器上报状态")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				app.openConfig()
			case <-mUpdate.ClickedCh:
				if err := app.UploadNow(); err != nil {
					log.Printf("手动上报失败: %v", err)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// openConfig 用默认浏览器打开本地配置页。
func (a *App) openConfig() {
	a.mu.RLock()
	listen := a.cfg.HTTP.Listen
	a.mu.RUnlock()
	openBrowser("http://" + listen + "/")
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

// makeIcon 运行时绘制一只可爱的小蓝脸图标（大眼、腮红、微笑、绿色在线点），
// 使用超采样抗锯齿，并包装为 PNG-in-ICO 格式（无需外部二进制资源）。
func makeIcon() []byte {
	return pngToICO(renderIconPNG())
}

// renderIconPNG 返回可爱图标（256x256 透明背景）的 PNG 编码字节。
func renderIconPNG() []byte {
	const size = 256 // 逻辑尺寸，足够清晰显示在任务栏
	const ss = 3     // 超采样倍数，减少锯齿
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a uint32
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(x) + (float64(sx) + 0.5) / float64(ss)
					fy := float64(y) + (float64(sy) + 0.5) / float64(ss)
					cr, cg, cb, ca := iconPixel(fx, fy, float64(size))
					r += uint32(cr)
					g += uint32(cg)
					b += uint32(cb)
					a += uint32(ca)
				}
			}
			n := uint32(ss * ss)
			out.SetRGBA(x, y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, out); err != nil {
		return nil
	}
	return pngBuf.Bytes()
}

// iconPixel 返回逻辑坐标 (x,y) 处的颜色（S 为画布边长，用于按比例计算各部件位置）。
func iconPixel(x, y, S float64) (uint8, uint8, uint8, uint8) {
	cx, cy := S/2, S/2
	bodyR := S * 0.44
	d := math.Hypot(x-cx, y-cy)
	if d > bodyR {
		return 0, 0, 0, 0
	}
	// 边缘描边（深蓝），让圆形更立体
	if d > bodyR-S*0.03 {
		return 37, 99, 235, 255
	}
	// 主体：柔和的蓝色
	col := color.RGBA{96, 165, 250, 255}

	// 腮红（粉色半透明）
	chkR := S * 0.075
	chkX := S * 0.27
	chkY := S * 0.08
	if insideCircle(x, y, cx-chkX, cy+chkY, chkR) || insideCircle(x, y, cx+chkX, cy+chkY, chkR) {
		col = blend(col, color.RGBA{255, 145, 175, 255}, 0.55)
	}

	// 眼睛（白底）
	eyeR := S * 0.105
	eyeDX := S * 0.17
	eyeY := S * 0.12
	if insideCircle(x, y, cx-eyeDX, cy-eyeY, eyeR) || insideCircle(x, y, cx+eyeDX, cy-eyeY, eyeR) {
		col = color.RGBA{255, 255, 255, 255}
	}
	// 瞳孔（深色，略偏下）
	pupR := S * 0.052
	pupY := S * 0.135
	if insideCircle(x, y, cx-eyeDX, cy-pupY, pupR) || insideCircle(x, y, cx+eyeDX, cy-pupY, pupR) {
		col = color.RGBA{30, 30, 45, 255}
	}
	// 瞳孔高光（小白点）
	glR := S * 0.02
	if insideCircle(x, y, cx-eyeDX-S*0.02, cy-pupY-S*0.022, glR) ||
		insideCircle(x, y, cx+eyeDX-S*0.02, cy-pupY-S*0.022, glR) {
		col = color.RGBA{255, 255, 255, 255}
	}

	// 嘴巴（微笑：下半圆）
	mR := S * 0.14
	mx, my := cx, cy+S*0.2
	if math.Hypot(x-mx, y-my) <= mR && y >= my {
		col = color.RGBA{30, 30, 45, 255}
	}

	// 在线状态点（绿色，右下角，位于主体圆内）
	dotR := S * 0.075
	dotX := S * 0.24
	dotY := S * 0.24
	if insideCircle(x, y, cx+dotX, cy+dotY, dotR) {
		if math.Hypot(x-(cx+dotX), y-(cy+dotY)) > dotR*0.72 { // 白边
			col = color.RGBA{255, 255, 255, 255}
		} else {
			col = color.RGBA{34, 197, 94, 255}
		}
	}
	return col.R, col.G, col.B, col.A
}

func insideCircle(x, y, cx, cy, r float64) bool {
	return math.Hypot(x-cx, y-cy) <= r
}

// blend 按 t 比例混合两种颜色。
func blend(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		a.A,
	}
}

func pngToICO(pngData []byte) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0}) // reserved
	buf.Write([]byte{1, 0}) // image type = icon
	buf.Write([]byte{1, 0}) // image count = 1
	// ICONDIRENTRY
	buf.WriteByte(0) // width (0 => 256)
	buf.WriteByte(0) // height
	buf.WriteByte(0) // colors
	buf.WriteByte(0) // reserved
	buf.Write([]byte{1, 0})                 // color planes
	buf.Write([]byte{32, 0})                // bits per pixel
	binary.Write(&buf, binary.LittleEndian, uint32(len(pngData)))
	binary.Write(&buf, binary.LittleEndian, uint32(6+16)) // offset of PNG data
	buf.Write(pngData)
	return buf.Bytes()
}
