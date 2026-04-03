# 底层驱动 (Drive Server)

## 说明

底层驱动是独立的服务，负责 QQ 协议通信，通过 WebSocket (端口 8098) 与主服务连接。

## 文件放置

请将驱动相关文件放入此目录：

```
drive-server/
├── README.md           ← 本说明文件
├── 驱动程序文件          ← 放这里（如 drive-server 可执行文件）
├── config/             ← 驱动配置文件（如有）
└── data/               ← 驱动运行时数据（如有）
```

## 主服务如何连接驱动

主服务通过环境变量 `DRIVE_MAPPING` 配置驱动地址：

```
格式: 硬件类型#软件类型#软件版本#驱动URL1#驱动URL2
示例: server1#androidQQ#8.9.80#http://127.0.0.1:8098
```

连接方式: HTTP 地址自动转换为 WebSocket → `ws://127.0.0.1:8098/websocket`

## 多驱动部署

支持多个驱动实例，用逗号分隔：

```
DRIVE_MAPPING=server1#androidQQ#8.9.80#http://10.0.0.1:8098,server2#iOSQQ#9.0.0#http://10.0.0.2:8098
```
