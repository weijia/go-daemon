package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	app, err := NewApp(p)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestHTTPServerConfigAPI(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/config")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config 状态 %d", resp.StatusCode)
	}
	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.UUID == "" {
		t.Fatal("返回的 uuid 为空")
	}

	// 修改间隔并保存
	cfg.Report.IntervalMinutes = 7
	body, _ := json.Marshal(cfg)
	r, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/config", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/config 状态 %d", resp2.StatusCode)
	}
	if got := app.Config().Report.IntervalMinutes; got != 7 {
		t.Fatalf("间隔未生效: %d", got)
	}
}

func TestHTTPServerStatusAndUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	app := newTestApp(t)
	app.mu.Lock()
	app.cfg.RemoteStorage.Server = ts.URL
	app.cfg.RemoteStorage.PathTemplate = "/{uuid}.json"
	app.cfg.RemoteStorage.Token = "t"
	app.mu.Unlock()

	srv := httptest.NewServer(app.routes())
	defer srv.Close()

	// 立即上报
	resp, err := http.Post(srv.URL+"/api/update", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/update 状态 %d", resp.StatusCode)
	}
	if !app.state.Snapshot().LastSuccess {
		t.Fatal("期望上报成功")
	}

	// 状态接口
	resp2, err := http.Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var st map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if _, ok := st["status"]; !ok {
		t.Fatal("状态接口缺少 status 字段")
	}
}

func TestHTTPServerInvalidConfig(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.routes())
	defer srv.Close()

	bad := Config{UUID: "", RemoteStorage: RemoteStorageConfig{Server: ""}, Report: ReportConfig{IntervalMinutes: 1}, HTTP: HTTPConfig{Listen: "127.0.0.1:1"}}
	body, _ := json.Marshal(bad)
	r, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/config", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法配置应返回 400，得到 %d", resp.StatusCode)
	}
}
