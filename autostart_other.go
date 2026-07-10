//go:build !windows

package main

// SetAutostart 在非 Windows 平台为 no-op（仅 Windows 有注册表 Run 自启）。
func SetAutostart(enabled bool) error { return nil }

// IsAutostartEnabled 非 Windows 平台始终返回 false。
func IsAutostartEnabled() bool { return false }
