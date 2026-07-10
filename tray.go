package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
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

// makeIcon 运行时绘制一个简单的蓝色圆形图标，并包装为 PNG-in-ICO 格式（无需外部二进制资源）。
func makeIcon() []byte {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	drawCircle(img, size/2, size/2, size/2-3, color.RGBA{37, 99, 235, 255})
	drawCircle(img, size/2, size/2, size/7, color.RGBA{255, 255, 255, 255})

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil
	}
	return pngToICO(pngBuf.Bytes())
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, c)
			}
		}
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
