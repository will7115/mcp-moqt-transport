# MCP over MOQT Transport

简体中文说明：本项目为 MCP over MOQT 的 Go 实现，提供面向 MCP SDK 的 `Transport`/`Connection` 适配层，用于在 QUIC + MOQT 上承载 MCP 的 JSON-RPC 消息。

## 开发说明

v0.1.2 的实现与文档更新仍由 AI 编写（OpenAI GPT-5，Codex），主要覆盖统一入口、TLS 默认行为、注释与示例更新；后续开发将由人工进行系统性重构和优化。

## 概述

该项目提供 MCP over MOQT 的最小可用传输层实现，目标是让上层只需 `server.Run(...)` / `client.Connect(...)` 即可连通。默认配置可用，并支持 Option 覆盖。

## 草案兼容性

- 当前实现基于 `moqtransport` 的 draft-11（ALPN `moq-00`）。
- 与 draft-16 的 stream/datagram 编码不兼容。

## 功能特性

- MCP 消息映射到 MOQT 控制轨道对象（control tracks）。
- discovery 通过 `mcp/discovery/sessions` 的 FETCH 完成。
- 默认配置可直接在本地开发环境跑通。

## 版本

当前版本：v0.1.2

## v0.1.2 更新概述

- 统一入口：`NewMoqTransport(RoleServer/RoleClient)`，上层保持 `Run/Connect` 的简洁调用方式。
- TLS 默认行为：server 端本地自签名证书；client 端默认 `InsecureSkipVerify`，均可通过 Option 覆盖。
- ALPN 默认 `moq-00`（基于 moqtransport draft-11）。
- discovery 仍使用 `mcp/discovery/sessions`（FETCH）。
- examples 已更新到统一入口；新增/补充草案注释标记。

## Roadmap / TODO

- 预留 `resources/tools/notifications` 轨道（Draft: draft-jennings-mcp-over-moqt-00 §2.3/§2.4）。
- v0.1.2 仅保留轨道命名与 TODO 注释，不实现具体业务语义；后续将补充订阅/发布与消息结构。

## 快速上手

1. `go get github.com/mcp-moqt/mcp-moqt-transport`
2. `go run ./examples/server -addr 127.0.0.1:8080`
3. `go run ./examples/client -addr 127.0.0.1:8080`

## 演示与预期结果

在两个终端分别运行示例：

终端 A（server）：
```bash
go run ./examples/server -addr 127.0.0.1:8080
```

终端 B（client）：
```bash
go run ./examples/client -addr 127.0.0.1:8080
```

预期输出：
- client 输出 `connected to 127.0.0.1:8080; ping ok`
- server 在 client 断开后返回 `context canceled`，这是单连接示例的正常行为

## 安装

```bash
go get github.com/mcp-moqt/mcp-moqt-transport
```

## 使用示例

### 服务端（QUIC + MOQT + MCP）

```go
package main

import (
    "context"
    "log"

    "github.com/mcp-moqt/mcp-moqt-transport"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    ctx := context.Background()

    transport, err := mcpmoqt.NewMoqTransport(
        mcpmoqt.RoleServer,
        mcpmoqt.WithAddr("127.0.0.1:8080"),
    )
    if err != nil {
        log.Fatalf("new transport: %v", err)
    }

    server := mcp.NewServer(&mcp.Implementation{
        Name:    "example-server",
        Version: "v0.1.2",
    }, nil)

    if err := server.Run(ctx, transport); err != nil {
        log.Fatalf("server run: %v", err)
    }
}
```

### 客户端（连接服务端并执行 MCP 调用）

```go
package main

import (
    "context"
    "log"

    "github.com/mcp-moqt/mcp-moqt-transport"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    ctx := context.Background()

    transport, err := mcpmoqt.NewMoqTransport(
        mcpmoqt.RoleClient,
        mcpmoqt.WithAddr("127.0.0.1:8080"),
    )
    if err != nil {
        log.Fatalf("new client transport: %v", err)
    }

    client := mcp.NewClient(&mcp.Implementation{
        Name:    "example-client",
        Version: "v0.1.2",
    }, nil)

    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        log.Fatalf("client connect: %v", err)
    }
    defer session.Close()

    if err := session.Ping(ctx, nil); err != nil {
        log.Fatalf("ping: %v", err)
    }
}
```

## 测试

### 本地端到端 MCP 测试

```bash
go test -v -run TestMCPServerClient_RunAndPing ./...
```

### 运行全部测试

```bash
go test -v ./...
```

### Docker 网络测试

```bash
docker-compose up --build
```

## 项目结构

```
mcp-moqt-transport/
transport.go        # MCP over MOQT 的 Transport/Connection 实现
server.go           # 服务端连接/会话封装
client.go           # 客户端连接/会话封装
options.go          # 可配置选项（addr/TLS/ALPN/QUIC）
tls.go              # TLS 默认行为与封装
quic_moq.go         # QUIC <-> moqtransport 适配
control_conn.go     # 控制轨道的 JSON-RPC 读写
session_handlers.go # discovery / subscribe handler
session_id.go       # MCP Session ID 生成
examples/           # 示例：server/client
connectivity_test.go# 端到端 Run + Ping 测试
README.md           # 项目说明
```

## 开发状态

- [x] Transport/Connection 适配 MCP go-sdk
- [x] MCP 消息映射为 MOQT 控制轨道对象
- [x] discovery（FETCH `mcp/discovery/sessions`）
- [x] QUIC + TLS 基础通路 + 本地连通测试
- [x] Options（`WithAddr`/`WithTLSServerConfig`/`WithTLSClientConfig`/`WithQUICConfig`）
- [ ] resources/tools/notifications 轨道与语义
- [ ] 更完整的错误模型与恢复策略

## 许可

MIT License

## 贡献

欢迎提交 Issue 或 Pull Request。
