# 芯添内网设备状态检测 (XinT-LanPulse)

一个面向 Windows 内网运维场景的设备状态检测工具，用于检测已登记设备的可达性，并通过桌面界面或飞书机器人查询结果。

本项目提供一套本地运行的设备巡检方案：

- 桌面端维护设备台账、发起检测并查看运行日志
- 使用 Windows 原生 ICMP Ping 检测 IPv4 设备
- 飞书机器人通过长连接接收私聊命令，无需配置公网入口
- 配置和设备数据保存在当前 Windows 用户目录

## 作者信息

- 原作者：`XinTycd`
- 项目名称：`XinT-LanPulse`
- 当前版本：`v0.3.0`
- 版权声明：[LICENSE](LICENSE)

## 项目特性

- 设备台账：添加、编辑和批量删除设备，维护 IP 后缀、名称与区域
- Excel 导入：导出模板并导入设备信息，减少重复录入
- 连通性检测：单设备检测、并发批量检测，可配置超时、尝试次数和并发数
- 定时巡检：按设定间隔检测设备列表并记录运行日志
- 飞书机器人：长连接接收私聊命令，支持查询权限与管理员权限
- 配置管理：备份和导入配置，使用 Windows DPAPI 加密保存飞书 App Secret
- 桌面运行：支持托盘、开机启动和 Windows 原生文件选择器
- 组件管理：内置设备状态检测与可选的监控设备增量包，可导入组件元数据

## 适用场景

- 办公室、机房和展厅的内网设备巡检
- 已登记设备的日常连通性查询
- 运维人员通过飞书私聊查询设备状态
- 需要在本机管理设备台账和定时检测的 Windows 工作站

## 技术架构

```text
Wails 桌面界面（HTML / CSS / JavaScript）
              │
              ▼
        Go 应用层 / Desktop API
          ├─ Windows ICMP 检测
          ├─ 设备台账、Excel 与定时任务
          ├─ 本地配置、日志与 DPAPI 凭证
          └─ 飞书长连接与命令处理
```

## 技术栈

### 桌面与后端

- 开发语言：`Go 1.26`
- 桌面框架：`Wails v2`、`WebView2`
- 连通性检测：Windows 原生 ICMP IPv4 API
- 机器人接入：飞书官方 Go SDK 长连接
- 表格处理：`excelize`
- 凭证保护：Windows DPAPI

### 前端

- 页面结构：`HTML5`
- 视觉样式：`CSS3`
- 交互逻辑：原生 `JavaScript`
- 前端静态文件：`cmd/xint-lanpulse/frontend/dist/`，由 Go 程序嵌入

## 目录结构

```text
.
├─ assets/                     # 应用与托盘图标
├─ cmd/xint-lanpulse/          # 桌面入口、API 与前端文件
├─ docs/                       # 配置、命令、构建和项目说明
├─ internal/
│  ├─ app/                    # 应用编排、定时任务与批量检测
│  ├─ assets/                 # Excel 导入与模板导出
│  ├─ checker/                # Windows ICMP 检测
│  ├─ command/                # 飞书命令解析与执行
│  ├─ components/             # 组件元数据解析
│  ├─ config/                 # 配置、校验与备份
│  ├─ feishu/                 # 消息处理与事件去重
│  ├─ security/               # 权限、限流与凭证加密
│  └─ transport/              # 飞书消息传输
├─ scripts/                    # 构建和清理脚本
├─ tests/                      # 自动化测试
├─ CHANGELOG.md
├─ go.mod
├─ LICENSE
└─ VERSION
```

## 运行方式

### 1. 环境要求

- Windows 系统及 Microsoft Edge WebView2 Runtime
- 与 `go.mod` 匹配的 Go 版本（当前为 Go 1.26）
- 如需飞书机器人，需准备飞书企业自建应用和可用的机器人权限

### 2. 运行测试

在项目根目录执行：

```powershell
go test ./...
```

### 3. 本地启动

```powershell
go run ./cmd/xint-lanpulse
```

首次启动后，可在桌面端设置默认网段和检测参数，再添加或导入设备。

### 4. 构建 Windows 客户端

```powershell
.\scripts\build-windows.ps1
```

构建输出为 `dist/XinT-LanPulse.exe`，脚本同时复制 `dist/icon.ico` 供托盘使用。图标资源更新方式见 [Windows 构建说明](docs/BUILD_WINDOWS.md)。

## 设备与网络配置

默认网段为 `192.168.20`。设备台账保存 IP 最后一段、设备名称和区域，实际检测地址由默认网段和设备后缀组合。例如默认网段为 `192.168.20`、后缀为 `15` 时，设备地址为 `192.168.20.15`。

Excel 导入模板包含 `设备IP`、`设备名称`、`设备区域` 三列；当前导入器要求 IP 为 `192.168.20.1` 至 `192.168.20.254`。

## 飞书机器人

在飞书开放平台创建企业自建应用，启用机器人，订阅“接收消息 v2.0”并选择长连接。客户端保存 App ID 和 App Secret 后，可在设置中启动连接。允许使用者需要加入 Open ID 查询名单；管理员也必须在查询名单中。

| 命令 | 用途 |
| --- | --- |
| `15` 或 `/ping 15` | 检测默认网段中的 `.15` |
| `/vi 15` | 检测设备并返回名称、区域及检测信息 |
| `/vi all` | 批量巡检设备列表 |
| `/vi no` | 批量巡检并只显示无法连接的设备 |
| `/20 15` | 临时检测 `192.168.20.15` |
| `/net` | 查看默认网段 |
| `/net 192.168.30` | 管理员修改默认网段 |
| `/status`、`/help` | 查看运行状态或命令帮助 |

`/vi` 当前固定使用 `192.168.20.x`；纯数字和 `/ping` 使用配置的默认网段。机器人仅处理私聊。完整配置步骤和命令见 [飞书配置](docs/FEISHU_SETUP.md) 与 [飞书命令](docs/COMMANDS.md)。

## 配置、数据与备份

首次运行时，程序在 `%APPDATA%\XinT-LanPulse` 下创建配置、日志和数据目录。飞书 App Secret 使用 Windows DPAPI 加密，与当前 Windows 用户绑定。

设置页可以导出和导入 JSON 备份。备份不包含飞书 App Secret；导入时保留本机设备身份与已有密钥。详见 [配置与数据](docs/CONFIGURATION.md)。

从 `v0.2.5` 升级时，可先导出 Excel 迁移表，再在 v0.3.0 设备台账中导入。迁移表仅支持 `192.168.20.1`～`192.168.20.254`，名称和区域需要补充；早期版本的 WebView 数据不会自动迁移。

## 当前功能边界

- 本工具检测已登记设备，不提供全网扫描、端口映射或内网穿透。
- 钉钉和企业微信当前仅保留设置项，尚未实现消息收发。
- 组件导入仅保存元数据，不执行第三方组件代码。
- 监控设备增量包目前只支持监控点管理，不采集或分析视频画面。
- 实际飞书联调和竞态测试尚未全面完成，并发访问存在待处理风险。

## 项目文档

- [项目上下文与开发说明](docs/PROJECT_CONTEXT.md)
- [飞书配置](docs/FEISHU_SETUP.md)
- [飞书命令](docs/COMMANDS.md)
- [配置与数据](docs/CONFIGURATION.md)
- [Windows 构建](docs/BUILD_WINDOWS.md)
- [更新记录](CHANGELOG.md)
