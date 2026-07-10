//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	autostartRegKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	autostartValueName = "UFSNode"
)

// exePathQuoted 返回带引号的当前可执行文件路径，避免路径含空格导致启动失败。
func exePathQuoted() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return `"` + filepath.Clean(p) + `"`, nil
}

// SetAutostart 在 Windows 注册表 Run 项下启用/禁用开机自启。
func SetAutostart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enabled {
		v, err := exePathQuoted()
		if err != nil {
			return err
		}
		return k.SetStringValue(autostartValueName, v)
	}
	// 禁用：删除键值（不存在时忽略）
	_ = k.DeleteValue(autostartValueName)
	return nil
}

// IsAutostartEnabled 检查当前是否已注册自启（且指向本程序）。
func IsAutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartRegKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(autostartValueName)
	if err != nil {
		return false
	}
	cur, err := exePathQuoted()
	if err != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(cur))
}
