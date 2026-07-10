package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configFileName = "config.json"

// RemoteStorageConfig 描述如何连接 RemoteStorage 服务器（标准 RemoteStorage 协议，Bearer Token 鉴权）。
type RemoteStorageConfig struct {
	Server       string `json:"server"`        // 存储根地址（含用户名路径），如 https://storage.5apps.com/weijia
	User         string `json:"user"`          // 存储用户段，用于路径模板中的 {user}
	Token        string `json:"token"`         // Bearer Token
	Scope        string `json:"scope"`         // 作用域，如 /ufs-nodes/（仅记录，便于阅读）
	PathTemplate string `json:"path_template"` // 路径模板，支持 {user} {uuid}，默认 /ufs-nodes/{uuid}.json
}

// ReportConfig 描述上报行为。
type ReportConfig struct {
	IntervalMinutes int  `json:"interval_minutes"` // 上报间隔（分钟）
	ExtraInfo       bool `json:"extra_info"`       // 是否附带本机基础信息
}

// HTTPConfig 描述本地配置服务监听地址。
type HTTPConfig struct {
	Listen string `json:"listen"` // 默认 127.0.0.1:9801
}

// Config 是程序完整配置，持久化于同目录 config.json。
type Config struct {
	UUID          string              `json:"uuid"`
	Name          string              `json:"name"`
	RemoteStorage RemoteStorageConfig `json:"remotestorage"`
	Report        ReportConfig        `json:"report"`
	HTTP          HTTPConfig          `json:"http"`
}

// generateUUID 使用 crypto/rand 生成 UUID v4。
func generateUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 极端情况下退回，理论上不会发生
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// defaultConfig 生成首次运行时的默认配置。
func defaultConfig() Config {
	host, _ := os.Hostname()
	if host == "" {
		host = "node"
	}
	return Config{
		UUID: generateUUID(),
		Name: host,
		RemoteStorage: RemoteStorageConfig{
			Server:       "https://remotestorage.example.com",
			User:         "me",
			Token:        "",
			Scope:        "/ufs-nodes/",
			PathTemplate: "/ufs-nodes/{uuid}.json",
		},
		Report: ReportConfig{IntervalMinutes: 15, ExtraInfo: true},
		HTTP:   HTTPConfig{Listen: "127.0.0.1:9801"},
	}
}

// ResolveURL 根据 server + path_template 计算 PUT 目标 URL，替换 {user}/{uuid} 占位符。
func (c *Config) ResolveURL() string {
	tmpl := c.RemoteStorage.PathTemplate
	if strings.TrimSpace(tmpl) == "" {
		tmpl = "/ufs-nodes/{uuid}.json"
	}
	repl := strings.NewReplacer("{user}", c.RemoteStorage.User, "{uuid}", c.UUID)
	path := repl.Replace(tmpl)
	base := strings.TrimRight(c.RemoteStorage.Server, "/")
	return base + path
}

// Validate 校验配置是否可用于上报。
func (c *Config) Validate() error {
	if c.UUID == "" {
		return fmt.Errorf("uuid 不能为空")
	}
	if c.RemoteStorage.Server == "" {
		return fmt.Errorf("remotestorage.server 不能为空")
	}
	if c.HTTP.Listen == "" {
		return fmt.Errorf("http.listen 不能为空")
	}
	if c.Report.IntervalMinutes <= 0 {
		return fmt.Errorf("report.interval_minutes 必须为正数")
	}
	return nil
}

// LoadConfig 从磁盘读取配置。
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// SaveConfig 原子写盘：先写临时文件再 rename，避免半写损坏。
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// ensureConfig 加载配置；不存在则生成默认配置；并对缺失字段补默认值。
func ensureConfig(path string) (Config, bool, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), true, nil
		}
		return Config{}, false, err
	}
	if cfg.UUID == "" {
		cfg.UUID = generateUUID()
	}
	if cfg.HTTP.Listen == "" {
		cfg.HTTP.Listen = "127.0.0.1:9801"
	}
	if cfg.Report.IntervalMinutes <= 0 {
		cfg.Report.IntervalMinutes = 15
	}
	if strings.TrimSpace(cfg.RemoteStorage.PathTemplate) == "" {
		cfg.RemoteStorage.PathTemplate = "/ufs-nodes/{uuid}.json"
	}
	if strings.TrimSpace(cfg.RemoteStorage.Scope) == "" {
		cfg.RemoteStorage.Scope = "/ufs-nodes/"
	}
	return cfg, false, nil
}
