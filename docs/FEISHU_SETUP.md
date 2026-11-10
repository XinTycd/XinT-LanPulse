# 飞书配置

在飞书开放平台创建**企业自建应用**，启用机器人并订阅“接收消息 v2.0”（`im.message.receive_v1`）事件。选择“使用长连接接收事件”，不要填写 Webhook URL。客户端使用官方 Go SDK 的 `larkws.NewClient()` 建立 WebSocket 长连接；无需公网 IP、端口映射或 Web 服务。

请仅订阅 v2.0 事件，不要同时订阅对应 v1.0 事件，以免收到重复消息。启动客户端且显示“已连接”后，再在开放平台保存长连接订阅配置并发布应用。机器人需要获得接收用户发给机器人消息的相应权限；群聊场景还需要添加机器人，并按企业规则授予群消息权限。

将可使用者的 Open ID（`ou_` 开头）逐行填入“允许查询用户”。管理员必须同时存在于允许查询名单中。

在“设置 → 平台接入”选择“飞书”并保存 App ID、App Secret 后，客户端会自动建立连接。钉钉、企业微信目前仅保留设置项与平台选择界面，尚未实现消息收发；需要机器人命令时请选择飞书。

官方参考：[服务端 SDK 概述](https://open.feishu.cn/document/server-docs/server-side-sdk)、[使用长连接接收事件](https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/request-url-configuration-case)。
