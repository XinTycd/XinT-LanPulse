# XinT-LanPulse 项目上下文

> 本文是为新对话、新开发者和开源协作准备的单一项目说明。修改功能前优先阅读本文、`README.md`、`go.mod` 与相关源码；不要根据旧的 Fyne 或 Relay 文档推断当前实现。

## 1. 项目定位

XinT-LanPulse 是 Windows 本地内网设备状态检测工具，不是网络扫描器。它针对已登记设备执行 ICMP Ping，供运维人员在桌面端或飞书机器人私聊中发起查询。

默认网段为 `192.168.20`。设备台账仅保存 IP 最后一段、设备名称和设备区域；实际 IP 由当前默认网段与最后一段组合。飞书 `/vi` 巡检命令目前为兼容监控场景，固定检测 `192.168.20.x`。

## 2. 当前功能状态

### 已实现

- Go + Wails v2 桌面应用；前端为静态 HTML/CSS/JavaScript，位于 `cmd/xint-lanpulse/frontend/dist`。
- Windows 原生 ICMP Ping、超时/重试配置、并发批量检测（默认 8 线程）。
- 设备增删改、批量删除、Excel 模板导出和 Excel 导入。
- 设备批量检测、设备定时检测、最近检测状态与运行日志。
- 飞书官方 Go SDK 长连接、事件去重、查询/管理员权限、限流和文本回复。
- 飞书命令：纯数字、`/ping`、`/vi`、`/net`、`/status`、`/help`；详情见 `COMMANDS.md`。
- 设备组件框架、组件包文件/URL 元数据导入、内置“内网设备状态检测”和可选“监控设备增量包”。
- 监控点仅能从设备列表加入；当前只完成监控点管理，不会采集或分析视频画面。
- 配置备份/导入、DPAPI 加密的飞书 Secret、开机启动、关闭至托盘、系统原生文件选择器。
- Windows 标题栏、任务栏与托盘使用同一透明 `assets/icon.ico`；资源组 ID 固定为 Wails 所需的 `3`。

### 已有界面/配置但尚未接通

- 钉钉与企业微信可在设置页选择和保存接入标识/密钥，但没有对应消息传输实现。生产可用机器人通道只有飞书。
- 外部组件导入当前只导入和保存组件元数据，未执行第三方组件代码。

### 明确不实现或尚未实现

- Relay 中转、跨电脑 `/dv` 指派、端口映射、内网穿透、全网扫描、远程命令执行。
- TCP/HTTP/UDP 探测、主动告警、视频/RTSP 截图、监控画面黑屏或冻结识别。

涉及画面采集、外部机器人平台、Relay 或外部组件执行时，先明确网络边界、凭证保护、权限、审计、数据保留和用户授权；不得直接接入生产凭证。

## 3. 关键目录

| 路径 | 作用 |
| --- | --- |
| `cmd/xint-lanpulse/main.go` | Wails 入口、托盘、窗口生命周期、CLI 辅助参数。 |
| `cmd/xint-lanpulse/desktop_api.go` | 前端可调用的桌面 API、设备/设置/备份/组件操作。 |
| `cmd/xint-lanpulse/frontend/dist` | 当前前端源文件（无前端构建链）。 |
| `internal/app` | 应用编排、运输层刷新、定时检测、批量 worker pool。 |
| `internal/checker` | Windows ICMP 实现。 |
| `internal/command` | 飞书文本命令解析与结果格式化。 |
| `internal/config` | 模型、校验、原子存储、备份导入。 |
| `internal/transport/feishu_direct` | 飞书长连接和消息发送。 |
| `internal/assets` | Excel 导入和模板导出。 |
| `internal/components` | 组件元数据包/URL 的解析与校验。 |
| `assets` | 应用/托盘图标源。 |
| `scripts/build-windows.ps1` | Windows GUI 构建与图标资源生成。 |
| `tests` | 单元与 Windows 集成测试。 |

## 4. 架构与数据流

```text
Wails 前端
  ↕ DesktopAPI
Application
  ├─ Config Store (%APPDATA%\\XinT-LanPulse)
  ├─ Windows ICMP Checker
  ├─ Command Service
  ├─ Logger / EventStore
  └─ Feishu Direct Transport ↔ 飞书机器人私聊
```

`config.Config` 的主要数据为应用身份、网络检测设置、飞书凭证与权限、平台选择、组件、设备台账、定时任务和监控点后缀。飞书 Secret 使用 DPAPI 加密；前端状态和配置备份不会回传明文 Secret。

## 5. 运行与构建

```powershell
# 使用项目 Go 模块测试
go test ./...

# 本地运行
go run ./cmd/xint-lanpulse

# 生成 Windows GUI 程序
.\scripts\build-windows.ps1
```

构建脚本使用 `-tags 'desktop,production'` 和 `-H=windowsgui`。若安装了 GNU `windres`，脚本会从 `cmd/xint-lanpulse/windows.rc` 重新生成标题栏图标资源；资源文件 `rsrc_windows_amd64.syso` 必须保留在源码中。

本项目在部分 Windows 环境中需要将 `GOPATH`、`GOMODCACHE` 与 `GOCACHE` 指向一个可写路径后再执行 Go 命令；这是本机 Go 环境问题，不应写死为应用运行时依赖。

## 6. 维护约束与常见陷阱

- 不要重新引入 Fyne：当前窗口和前端均由 Wails 承载，旧 Fyne 代码已删除。
- Wails 读取 Windows 图标资源组 ID `3`。更新图标要同步 `assets/icon.ico`、`windows.rc`、`rsrc_windows_amd64.syso`，并重新构建。
- 关闭窗口默认隐藏至托盘；只有托盘菜单“退出”才允许完全结束程序。
- 批量检测必须使用 `Network.ConcurrentWorkers`，不要在 UI 线程内串行阻塞。
- `Application.GetState` 返回的数组必须保持非 `nil`，避免前端读取 `.length` 时崩溃。
- 配置保存会重新初始化运输层和定时任务。修改配置模型后必须同步校验、备份导入、桌面 API 和前端表单。
- 删除设备时必须同步清理 `CameraAssetSuffixes`，批量删除亦然。
- 根目录 `dist` 是构建产物，禁止提交；`cmd/xint-lanpulse/frontend/dist` 是当前嵌入的前端源文件，必须提交。`go.sum` 是依赖锁定文件，必须提交。

## 7. 开源前建议

- 初始化 Git 仓库并提交源码（包括嵌入的前端文件）、`go.sum`、图标资源与文档；不要提交根目录 `dist`、`%APPDATA%` 配置、日志或任何 App Secret。
- 为飞书、设备批量检测、备份导入和组件解析保留/扩展自动化测试。
- 实现钉钉或企业微信前，先建立对应 transport、事件验证、权限模型、重试和测试；不能只把配置项标记为“已连接”。
