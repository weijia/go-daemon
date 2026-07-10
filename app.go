package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// ReportResult 是 /api/status 返回的精简上报状态。
type ReportResult struct {
	LastAttempt time.Time `json:"last_attempt"`
	LastSuccess bool      `json:"last_success"`
	LastError   string    `json:"last_error,omitempty"`
}

// ReportState 记录最近一次上报结果，并发安全。
type ReportState struct {
	mu          sync.RWMutex
	LastAttempt time.Time
	LastSuccess bool
	LastError   string
}

func (r *ReportState) Set(success bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.LastAttempt = time.Now()
	r.LastSuccess = success
	if err != nil {
		r.LastError = err.Error()
	} else {
		r.LastError = ""
	}
}

func (r *ReportState) Snapshot() ReportResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return ReportResult{
		LastAttempt: r.LastAttempt,
		LastSuccess: r.LastSuccess,
		LastError:   r.LastError,
	}
}

// App 是程序的中心状态：持有配置、上报状态、HTTP 服务与定时上报循环。
type App struct {
	cfgPath   string
	mu        sync.RWMutex
	cfg       Config
	client    *http.Client
	state     ReportState
	httpMu    sync.Mutex
	httpSrv   *http.Server
	retrigger chan struct{}
	quit      chan struct{}
	quitOnce  sync.Once
}

// NewApp 加载或生成配置，并构造 App。
func NewApp(cfgPath string) (*App, error) {
	cfg, created, err := ensureConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	if created {
		if err := SaveConfig(cfgPath, cfg); err != nil {
			return nil, err
		}
	}
	app := &App{
		cfgPath:   cfgPath,
		cfg:       cfg,
		client:    &http.Client{Timeout: 10 * time.Second},
		retrigger: make(chan struct{}, 1),
		quit:      make(chan struct{}),
	}
	app.applyAutostart()
	return app, nil
}

// applyAutostart 将当前配置中的 autostart 状态同步到系统（如 Windows 注册表 Run 项）。
func (a *App) applyAutostart() {
	a.mu.RLock()
	enabled := a.cfg.Autostart
	a.mu.RUnlock()
	if err := SetAutostart(enabled); err != nil {
		log.Printf("设置开机自启失败: %v", err)
	}
}

// Config 返回当前配置的副本。
func (a *App) Config() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *App) interval() time.Duration {
	a.mu.RLock()
	m := a.cfg.Report.IntervalMinutes
	a.mu.RUnlock()
	if m <= 0 {
		m = 15
	}
	return time.Duration(m) * time.Minute
}

// UploadNow 立即上报一次，并记录结果。
func (a *App) UploadNow() error {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()
	err := Upload(cfg, a.client)
	a.state.Set(err == nil, err)
	return err
}

// ApplyConfig 校验、持久化新配置，并重启定时上报（间隔变化时）；若监听地址变化则优雅重启 HTTP 服务。
func (a *App) ApplyConfig(newCfg Config) error {
	if err := newCfg.Validate(); err != nil {
		return err
	}
	a.mu.Lock()
	oldListen := a.cfg.HTTP.Listen
	a.cfg = newCfg
	a.mu.Unlock()

	if err := SaveConfig(a.cfgPath, newCfg); err != nil {
		return err
	}

	// 同步开机自启状态（如 Windows 注册表）
	a.applyAutostart()

	// 触发定时循环以应用新的间隔
	select {
	case a.retrigger <- struct{}{}:
	default:
	}

	if newCfg.HTTP.Listen != oldListen {
		a.restartHTTP()
	}
	return nil
}

// reportLoop 启动即上报一次，之后按配置间隔循环上报。
func (a *App) reportLoop() {
	a.UploadNow()
	for {
		timer := time.NewTimer(a.interval())
		select {
		case <-a.quit:
			timer.Stop()
			return
		case <-a.retrigger:
			timer.Stop()
			continue
		case <-timer.C:
			a.UploadNow()
		}
	}
}

// Start 启动上报循环与本地 HTTP 服务（均为后台 goroutine）。
func (a *App) Start() {
	go a.reportLoop()
	go a.listenHTTP()
}

// listenHTTP 在配置地址上启动 HTTP 服务。
func (a *App) listenHTTP() {
	a.mu.RLock()
	listen := a.cfg.HTTP.Listen
	a.mu.RUnlock()
	srv := &http.Server{
		Addr:         listen,
		Handler:      a.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	a.httpMu.Lock()
	a.httpSrv = srv
	a.httpMu.Unlock()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP 服务错误: %v", err)
	}
}

// restartHTTP 优雅关闭当前 HTTP 服务并以最新配置重启。
func (a *App) restartHTTP() {
	a.httpMu.Lock()
	old := a.httpSrv
	a.httpMu.Unlock()
	if old != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = old.Shutdown(ctx)
		cancel()
	}
	go a.listenHTTP()
}

// Quit 释放资源：关闭上报循环与 HTTP 服务。
func (a *App) Quit() {
	a.quitOnce.Do(func() {
		close(a.quit)
		a.httpMu.Lock()
		srv := a.httpSrv
		a.httpMu.Unlock()
		if srv != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_ = srv.Shutdown(ctx)
			cancel()
		}
	})
}
