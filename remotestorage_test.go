package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildPayload(t *testing.T) {
	c := defaultConfig()
	c.Report.ExtraInfo = true
	b, err := BuildPayload(c)
	if err != nil {
		t.Fatal(err)
	}
	var p NodePayload
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	if p.UUID != c.UUID || p.Name != c.Name || p.Status != "online" || p.LastAccess == 0 {
		t.Fatalf("payload 异常: %+v", p)
	}
	if p.OS == "" || p.Version == "" {
		t.Fatalf("缺少本机信息: %+v", p)
	}

	c.Report.ExtraInfo = false
	b2, _ := BuildPayload(c)
	var p2 NodePayload
	_ = json.Unmarshal(b2, &p2)
	if p2.OS != "" || p2.Version != "" {
		t.Fatalf("extra_info=false 时不应包含本机信息: %+v", p2)
	}
}

func TestUploadClassifies(t *testing.T) {
	cases := []struct {
		code     int
		wantKind string
		success  bool
	}{
		{http.StatusOK, "", true},
		{http.StatusCreated, "", true},
		{http.StatusUnauthorized, "auth", false},
		{http.StatusForbidden, "auth", false},
		{http.StatusInternalServerError, "server", false},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("期望 PUT，得到 %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer tok" {
				t.Error("缺少 Bearer 鉴权头")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("缺少 Content-Type")
			}
			w.WriteHeader(tc.code)
		}))
		cfg := Config{
			UUID: "u", Name: "n",
			RemoteStorage: RemoteStorageConfig{Server: srv.URL, User: "u", Token: "tok", PathTemplate: "/{uuid}.json"},
		}
		err := Upload(cfg, srv.Client())
		srv.Close()
		if tc.success {
			if err != nil {
				t.Fatalf("code %d: 期望成功，得到 %v", tc.code, err)
			}
			continue
		}
		ue, ok := err.(*UploadError)
		if !ok {
			t.Fatalf("code %d: 期望 *UploadError，得到 %T", tc.code, err)
		}
		if ue.Kind != tc.wantKind {
			t.Fatalf("code %d: 期望 kind=%q，得到 %q", tc.code, tc.wantKind, ue.Kind)
		}
	}
}

func TestUploadNetworkError(t *testing.T) {
	cfg := Config{UUID: "u", RemoteStorage: RemoteStorageConfig{Server: "http://127.0.0.1:1", PathTemplate: "/{uuid}.json"}}
	err := Upload(cfg, http.DefaultClient)
	if err == nil {
		t.Fatal("期望网络错误")
	}
	ue, ok := err.(*UploadError)
	if !ok || ue.Kind != "network" {
		t.Fatalf("网络错误分类错误: %v", err)
	}
}
