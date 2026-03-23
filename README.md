# XinT-LanPulse v0.1.4（重建版）

Windows 内网连通性检测雏形，支持 IPv4 地址检测、自定义超时和尝试次数，显示最近 20 条本次检测记录，并支持维护本地设备地址清单。

需要 Go 1.26 与 Windows WebView2。运行 `go test ./...` 验证，运行 `go run ./cmd/xint-lanpulse` 启动。

本版本根据现有源码重建，并非从原始 Git 历史恢复。设备清单、地址和检测参数保存在当前用户的 WebView 数据中，升级至其他实现时不会自动迁移。
