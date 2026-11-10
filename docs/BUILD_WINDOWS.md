# Windows 构建

项目使用 Go、Wails v2 和 WebView2。安装与 `go.mod` 匹配版本的 Go 后，在项目根目录运行：

```powershell
.\scripts\build-windows.ps1
```

输出为 `dist\\XinT-LanPulse.exe`，同时会复制 `dist\\icon.ico` 供托盘使用。

`cmd/xint-lanpulse/rsrc_windows_amd64.syso` 是标题栏图标资源，已纳入源码。若替换 `assets/icon.ico`，需要安装 GNU `windres` 并重新运行构建脚本，以便生成 Wails 所需的 Windows 资源组 ID 3。
