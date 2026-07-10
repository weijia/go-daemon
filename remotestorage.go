package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

// version 由编译期 -ldflags "-X main.version=..." 注入，默认 dev。
var version = "dev"

// UploadError 对上报失败按类别归类，便于在状态页/日志中区分处理。
type UploadError struct {
	Kind string // "auth" | "network" | "server" | "invalid"
	Err  error
}

func (e *UploadError) Error() string { return fmt.Sprintf("%s: %v", e.Kind, e.Err) }
func (e *UploadError) Unwrap() error { return e.Err }

// NodePayload 是 PUT 到 RemoteStorage 的节点 JSON 结构，符合 UFS Nodes 规范并附带基础本机信息。
type NodePayload struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	LastAccess int64  `json:"last_access"`
	Hostname   string `json:"hostname,omitempty"`
	OS         string `json:"os,omitempty"`
	Version    string `json:"version,omitempty"`
}

// BuildPayload 依据配置构建上报的 JSON 字节。
func BuildPayload(cfg Config) ([]byte, error) {
	p := NodePayload{
		UUID:       cfg.UUID,
		Name:       cfg.Name,
		Status:     "online",
		LastAccess: time.Now().UnixMilli(),
	}
	if cfg.Report.ExtraInfo {
		host, _ := os.Hostname()
		p.Hostname = host
		p.OS = runtime.GOOS
		p.Version = version
	}
	return json.Marshal(p)
}

// Upload 将本节点状态 PUT 到 RemoteStorage 服务器，按状态码归类错误。
func Upload(cfg Config, client *http.Client) error {
	if client == nil {
		client = http.DefaultClient
	}
	target := cfg.ResolveURL()
	if target == "" {
		return &UploadError{Kind: "invalid", Err: fmt.Errorf("目标 URL 为空")}
	}
	body, err := BuildPayload(cfg)
	if err != nil {
		return &UploadError{Kind: "invalid", Err: err}
	}
	req, err := http.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	if err != nil {
		return &UploadError{Kind: "invalid", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.RemoteStorage.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.RemoteStorage.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return &UploadError{Kind: "network", Err: err}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return &UploadError{Kind: "auth", Err: fmt.Errorf("服务器返回 %d（请检查 Token）", resp.StatusCode)}
	default:
		return &UploadError{Kind: "server", Err: fmt.Errorf("服务器返回 %d", resp.StatusCode)}
	}
}
