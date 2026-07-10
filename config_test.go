package main

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	u1 := generateUUID()
	if !re.MatchString(u1) {
		t.Fatalf("无效的 UUID v4: %s", u1)
	}
	u2 := generateUUID()
	if u1 == u2 {
		t.Fatal("两次生成的 UUID 不应相同")
	}
}

func TestResolveURL(t *testing.T) {
	c := Config{
		UUID: "abc",
		RemoteStorage: RemoteStorageConfig{
			Server:       "https://example.com/",
			User:         "me",
			PathTemplate: "/storage/{user}/ufs-nodes/{uuid}.json",
		},
	}
	if got, want := c.ResolveURL(), "https://example.com/storage/me/ufs-nodes/abc.json"; got != want {
		t.Fatalf("ResolveURL = %q, want %q", got, want)
	}

	c2 := Config{UUID: "x", RemoteStorage: RemoteStorageConfig{Server: "https://s.com", PathTemplate: ""}}
	if got := c2.ResolveURL(); got != "https://s.com/ufs-nodes/x.json" {
		t.Fatalf("默认模板 ResolveURL = %q", got)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	cfg := defaultConfig()
	cfg.Name = "test-node"
	if err := SaveConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "test-node" || loaded.UUID != cfg.UUID {
		t.Fatalf("往返不一致: %+v", loaded)
	}
}

func TestEnsureConfigCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	cfg, created, err := ensureConfig(p)
	if err != nil || !created {
		t.Fatalf("err=%v created=%v", err, created)
	}
	if cfg.UUID == "" || cfg.HTTP.Listen == "" || cfg.Report.IntervalMinutes <= 0 {
		t.Fatalf("默认值不完整: %+v", cfg)
	}
	// 首次生成的配置需落盘后，再次加载才应视为已存在
	if err := SaveConfig(p, cfg); err != nil {
		t.Fatal(err)
	}
	if _, created2, _ := ensureConfig(p); created2 {
		t.Fatal("已存在配置不应被重新创建")
	}
}

func TestValidate(t *testing.T) {
	c := defaultConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("默认配置应合法: %v", err)
	}
	c.RemoteStorage.Server = ""
	if err := c.Validate(); err == nil {
		t.Fatal("空的 server 应校验失败")
	}
	c = defaultConfig()
	c.Report.IntervalMinutes = 0
	if err := c.Validate(); err == nil {
		t.Fatal("间隔为 0 应校验失败")
	}
}

func TestConfigJSONFields(t *testing.T) {
	c := defaultConfig()
	b, _ := json.Marshal(c)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"uuid", "name", "remotestorage", "report", "http"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("序列化缺少字段 %q", k)
		}
	}
}
