package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"image"
	"image/png"
	"log"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

//go:embed appicon.svg
var appIconSVG []byte

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

// makeIcon 从内嵌的 SVG 源渲染托盘图标：渲染为 256x256 PNG，再包成 ICO
// 容器（Windows 的 systray 需要 ICO 格式；Linux/macOS 也能识别）。
func makeIcon() []byte {
	const size = 256
	icon, err := oksvg.ReadIconStream(bytes.NewReader(appIconSVG))
	if err != nil {
		log.Printf("解析 SVG 图标失败，使用空白托盘图标: %v", err)
		return nil
	}
	icon.SetTarget(0, 0, size, size)
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, rgba); err != nil {
		log.Printf("编码图标 PNG 失败: %v", err)
		return nil
	}
	return pngToICO(pngBuf.Bytes())
}

// pngToICO 将 PNG 字节包成单帧 ICO 容器（reserved/type/count + ICONDIRENTRY + PNG 数据）。
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
