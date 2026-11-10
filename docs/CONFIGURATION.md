# 配置与数据

首次运行将在 `%APPDATA%\\XinT-LanPulse` 创建 `config.json`、`logs` 和 `data`。飞书 App Secret 使用 Windows DPAPI 加密，不能跨 Windows 用户或电脑复制使用。配置写入使用同目录临时文件替换，损坏配置会备份为带时间戳的 `.broken` 文件。

“设置”页提供配置备份和导入。备份是 JSON 文件，包含网络、设备列表、权限、组件、定时任务和程序设置，但**不包含** App Secret。导入时始终保留本机的设备 ID、设备名称及已有 App Secret，避免把一台电脑的身份或 DPAPI 密文复制到另一台电脑。
