# XinT-LanPulse

> 这是依据现有源码重建的版本序列；历史提交日期为整理版本时指定，并非原始开发或推送日期。

从重建版 v0.2.5 升级时，可先导出 Excel 迁移表，再通过设备台账导入；仅支持 192.168.20.1～254 的设备地址，名称和区域需在此版本补充。此前 WebView 保存的数据不会自动迁移。

XinT-LanPulse 是面向 Windows 内网运维场景的本地设备状态检测工具。它以原生 ICMP Ping 检测设备可达性，通过飞书机器人接收巡检命令，并提供设备台账、Excel 导入、定时检测、托盘运行和本地配置备份。

当前桌面端基于 **Go + Wails v2 + WebView2**，界面为轻量的深色工具风格；敏感凭证以 Windows DPAPI 加密后保存到当前用户目录。

## 功能

- 设备列表：维护设备 IP、名称、设备区域，支持添加、编辑、批量删除和 Excel 导入。
- 连通性检测：临时检测、设备批量检测、可配置超时、重试次数和并发线程数。
- 飞书机器人：长连接接收 `/vi`、`/ping`、`/net`、`/status` 等命令，并按查询/管理员权限控制。
- 定时检测：按设定间隔巡检设备列表并写入运行日志。
- 组件：内置“内网设备状态检测”；启用“监控设备增量包”后可管理监控点。
- 设置与运维：配置备份/导入、开机启动、关闭至托盘、系统原生文件选择器。

## 快速开始

```powershell
go test ./...
go run ./cmd/xint-lanpulse
```

构建 Windows 客户端：

```powershell
.\scripts\build-windows.ps1
```

更多信息：

- [项目上下文与开发说明](docs/PROJECT_CONTEXT.md)
- [飞书配置](docs/FEISHU_SETUP.md)
- [飞书命令](docs/COMMANDS.md)
- [配置与数据](docs/CONFIGURATION.md)
- [Windows 构建](docs/BUILD_WINDOWS.md)
