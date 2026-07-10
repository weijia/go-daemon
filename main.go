package main

import (
	"log"
	"os"
	"path/filepath"
)

// configPath 返回与可执行文件同目录的 config.json 路径。
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return configFileName
	}
	return filepath.Join(filepath.Dir(exe), configFileName)
}

func main() {
	path := configPath()
	app, err := NewApp(path)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
	app.Start()
	runTray(app)
}
