# UFS Node — RemoteStorage 上报代理

一个纯 Go、无 CGO 的 Windows 系统栏（托盘）常驻程序。它仅有一个托盘图标，按配置间隔将本节点状态按 **UFS Nodes RemoteStorage 格式**上报到兼容 RemoteStorage 协议的服务器；所有配置保存在同目录的 `config.json` 中，并可通过本地 HTTP 服务（网页 + REST API）查看与修改。空闲时几乎零 CPU 占用，编译为单一静态 `.exe`。

## 特性

- 仅一个系统托盘图标：右键菜单可「打开配置」「立即上报」「退出」。
- 周期性上报：默认每 15 分钟（可配置）向 RemoteStorage 服务器 `PUT` 节点 JSON；启动即上报一次。
- 符合规范的数据格式：`{ uuid, name, status, last_access }`，并默认附带本机基础信息（hostname / 操作系统 / 版本）。
- 标准 RemoteStorage 协议：`Authorization: Bearer <token>` 鉴权，目标 URL = `server`（**存储根地址，含用户名路径**）拼接 `path_template`；默认 `{server}/ufs-nodes/{uuid}.json`（路径模板可配置）。
- 本地配置持久化：首次运行自动生成 `config.json`（含自动生成的 UUID v4、默认取主机名）。
- 本地 HTTP 配置：浏览器打开即看内置配置页；同时提供 REST API，便于脚本集成。
- 极低资源：仅 3 个常驻 goroutine（托盘、HTTP 服务、定时上报），仅在定时或手动触发时才发起网络请求。

## 配置字段（config.json）

| 字段 | 说明 |
|------|------|
| `uuid` | 节点唯一标识（UUID v4），自动生成，请勿随意修改 |
| `name` | 节点显示名称 |
| `autostart` | 是否开机自启（默认 `true`）；勾选后写入 Windows 注册表 Run 项，登录后自动运行（仅 Windows 生效） |
| `remotestorage.server` | 存储根地址（**含用户名路径**），如 `https://storage.5apps.com/weijia` |
| `remotestorage.user` | 存储用户段，用于路径模板中的 `{user}` |
| `remotestorage.token` | Bearer Token |
| `remotestorage.scope` | 作用域（仅记录，便于阅读），如 `/ufs-nodes/` |
| `remotestorage.path_template` | 路径模板（相对存储根），支持 `{user}` `{uuid}`，默认 `/ufs-nodes/{uuid}.json` |
| `report.interval_minutes` | 上报间隔（分钟），默认 15 |
| `report.extra_info` | 是否附带本机基础信息（hostname / os / version） |
| `http.listen` | 本地 HTTP 监听地址（仅本机），默认 `127.0.0.1:9801` |

目标 URL 计算：去掉 `server` 结尾的 `/` 后拼接 `path_template`（替换 `{user}`/`{uuid}`）。`server` 即存储根，例如 5apps 为 `https://storage.5apps.com/weijia`，最终 PUT 到 `https://storage.5apps.com/weijia/ufs-nodes/<uuid>.json`。

> 注意：`server` 应填**存储根**（已包含用户名路径），不要把用户段同时写进 `server` 和 `path_template` 的 `{user}`，否则路径会被重复拼接。

## 本地 HTTP 接口

仅绑定 `127.0.0.1`，外部不可访问。

- `GET  /`              内置配置页（HTML 表单）
- `GET  /api/config`    返回当前配置 JSON（含 token，便于表单回填）
- `POST /api/config`    应用并持久化新配置（间隔变化即时生效；监听地址变化自动重启 HTTP 服务）
- `GET  /api/status`    返回配置摘要与最近一次上报结果
- `POST /api/update`    立即触发一次上报

示例（REST）：

```bash
# 查看状态
curl http://127.0.0.1:9801/api/status

# 修改上报间隔为 5 分钟
curl -X POST http://127.0.0.1:9801/api/config \
  -H 'Content-Type: application/json' \
  -d '{"uuid":"<现有uuid>","name":"我的节点","remotestorage":{"server":"https://storage.5apps.com/weijia","user":"weijia","token":"TOKEN","scope":"/ufs-nodes/","path_template":"/ufs-nodes/{uuid}.json"},"report":{"interval_minutes":5,"extra_info":true},"http":{"listen":"127.0.0.1:9801"}}'
```

> 提示：先用 `GET /api/config` 取回当前完整配置，再按需修改后 `POST` 回去，可避免遗漏字段。

## 运行

直接双击 `go-daemon.exe` 即可。首次运行会在同目录生成 `config.json`，并在系统托盘出现图标。右键图标 → 「打开配置」在浏览器中填写 RemoteStorage 服务器信息并保存。

## 构建

需要 Go 1.21+。本项目无 CGO，交叉编译为单一 Windows 可执行文件：

```bash
# 在本机（Windows）直接构建；-H=windowsgui 使运行时不弹出 console 窗口
go build -ldflags "-H=windowsgui -X main.version=0.1.0" -o go-daemon.exe .

# 在其他平台交叉编译到 Windows（纯静态、无 CGO）
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -ldflags "-H=windowsgui -X main.version=0.1.0" -o go-daemon.exe .
```

> 推送 `v*.*.*` 形式的 tag（例如 `v0.1.0`）会自动触发 GitHub Actions：
> 构建 Windows exe 并发布带 `go-daemon.exe` 的 Release。

构建产物为单一 `go-daemon.exe`，不依赖任何外部 DLL 或运行时。

## 测试

```bash
go test -count=1 ./...
```

覆盖：UUID 生成、URL 解析、配置读写与默认值、上报 payload、错误分类（auth/network/server）、HTTP 接口、托盘图标生成。

## 资源占用

- 编译后单文件约 10 MB。
- 常驻：1 个托盘 goroutine + 1 个 HTTP 服务 goroutine（空闲近 0 CPU）+ 1 个定时上报 goroutine（分钟级触发）。
- 仅在定时/手动上报时发起一次 `PUT`；`http.Client` 复用连接并设置 10s 超时。
- 配置写盘采用「临时文件 + rename」原子写入，避免半写损坏。
