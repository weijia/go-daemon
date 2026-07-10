package main

import (
	"encoding/json"
	"net/http"
)

// routes 构建本地 HTTP 服务的路由（仅绑定 127.0.0.1）。
func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/api/config", a.handleConfig)
	mux.HandleFunc("/api/status", a.handleStatus)
	mux.HandleFunc("/api/update", a.handleUpdate)
	return mux
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(configPageHTML))
}

// handleConfig: GET 返回当前配置；POST 应用新配置。
func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.Config())
	case http.MethodPost:
		var newCfg Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "配置 JSON 解析失败: " + err.Error()})
			return
		}
		if err := a.ApplyConfig(newCfg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "config": a.Config()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStatus 返回节点配置摘要与最近一次上报结果。
func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"config": a.Config(),
		"status": a.state.Snapshot(),
	})
}

// handleUpdate 立即触发一次上报。
func (a *App) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := a.UploadNow()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error(), "status": a.state.Snapshot()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "status": a.state.Snapshot()})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
