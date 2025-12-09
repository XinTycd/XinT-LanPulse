# XinT-LanPulse v0.1.0（重建版）

Windows 内网连通性检测雏形。输入 IPv4 地址，使用 Windows 原生 ICMP 检测一次，显示结果。

需要 Go 1.26 与 Windows WebView2。运行 `go test ./...` 验证，运行 `go run ./cmd/xint-lanpulse` 启动。

本版本根据现有源码重建，并非从原始 Git 历史恢复。尚不包含设备台账、飞书、定时任务或配置保存。
