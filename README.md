# MCP over MOQT Transport

MCP over MOQT Transport 是一个实现了 Model Context Protocol (MCP) over Media over QUIC Transport (MOQT) 的 Go 语言传输层实现。

## 开发说明

本项目使用 [Cursor](https://cursor.sh/) 进行快速原型开发和思路验证。使用 Cursor 编写代码的目的是为了**快速验证开发思路和架构设计**，**不代表代码本身的开发质量情况**。

我们的团队承诺将**持续依据 IETF 草案独立进行标准化开发**，确保代码质量、性能优化、安全性以及符合相关标准规范。后续开发将遵循标准的软件工程实践，包括但不限于：

- 完整的单元测试和集成测试
- 代码审查和质量保证流程
- 性能优化和安全性审计
- 完整的文档和示例代码
- 符合 IETF 草案规范的实现

## 概述

本项目实现了 [draft-jennings-mcp-over-moqt-00](https://datatracker.ietf.org/doc/draft-jennings-mcp-over-moqt/) 草案中定义的 MCP over MOQT 传输协议，将 MCP 消息映射到 MOQT 对象，实现高效的发布-订阅通信。

## 功能特性

- ✅ 基本的对象映射为 MOQT 的载荷
- ✅ 控制轨道（Control Tracks）实现
- ✅ 客户端和服务器端传输实现
- ✅ 会话发现机制
- ✅ 连通性测试支持

## 版本

当前版本: v0.1.1

## 安装

```bash
go get github.com/mcp-moqt/mcp-moqt-transport
```

## 使用示例

### 服务器端（QUIC + MOQT + MCP）

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

    // 1. 准备 TLS（示例使用内存自签名证书，仅适合本地测试）
    tlsCfg, err := mcpmoqt.SelfSignedTLSServerConfig()
    if err != nil {
        log.Fatalf("tls: %v", err)
    }

    // 2. 通过选项（options）创建基于 QUIC 的 MOQT 传输
    transport, err := mcpmoqt.NewMOQTServerTransport(
        mcpmoqt.WithAddr("127.0.0.1:8080"),
        mcpmoqt.WithTLSServerConfig(tlsCfg),
    )
    if err != nil {
        log.Fatalf("new transport: %v", err)
    }

    // 3. 创建 MCP 服务器并直接使用 Run 运行
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "example-server",
        Version: "v0.1.1",
    }, nil)

    if err := server.Run(ctx, transport); err != nil {
        log.Fatalf("server run: %v", err)
    }
}
```

### 客户端（连接服务器并执行 MCP 调用）

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

    // 1. 通过选项创建客户端传输（内部完成 QUIC + MOQT Session 建立）
    transport, err := mcpmoqt.NewMOQTClientTransport(
        mcpmoqt.WithAddr("127.0.0.1:8080"),
    )
    if err != nil {
        log.Fatalf("new client transport: %v", err)
    }

    // 2. 创建 MCP 客户端
    client := mcp.NewClient(&mcp.Implementation{
        Name:    "example-client",
        Version: "v0.1.1",
    }, nil)

    // 3. 通过 MCP 的 Connect 建立会话
    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        log.Fatalf("client connect: %v", err)
    }
    defer session.Close()

    // 4. 做一次 ping 验证链路
    if err := session.Ping(ctx, nil); err != nil {
        log.Fatalf("ping: %v", err)
    }
}
```

## 测试

### 本地端到端 MCP 测试

当前版本提供了一个端到端测试，用于验证：

- QUIC 监听 / 拨号是否正常
- MOQT Session 是否能正确建立
- 会话发现（discovery / Fetch）是否工作
- MCP `Server.Run` / `Client.Connect` 能否在 MOQT 之上跑通 `Ping`

运行：

```bash
go test -v -run TestMCPServerClient_RunAndPing ./...
```

该测试会：

- 随机选择本地端口
- 启动基于 `NewMOQTServerTransport` 的 MCP 服务器
- 使用 `NewMOQTClientTransport` 连接服务器
- 完成一次 `Ping` 调用并验证无错误返回

### 运行所有测试

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
├── transport.go        # MCP over MOQT 的公共接口与兼容层
├── server.go           # 服务器端传输实现（基于 options + 内部 QUIC/MOQT Session）
├── client.go           # 客户端传输实现（基于 options + discovery）
├── options.go          # 传输配置（addr、QUIC/TLS、ALPN 等）
├── tls.go              # 本地测试用 TLS 自签名配置
├── quic_moq.go         # QUIC <-> moqtransport 适配（quic-go / quicmoq）
├── control_conn.go     # MCP Connection 封装（基于控制轨道的 JSON-RPC）
├── session_handlers.go # MOQT Handler / SubscribeHandler / discovery 处理
├── session_id.go       # MCP Session ID 生成
├── examples/           # 服务端 / 客户端示例
├── connectivity_test.go# 端到端 MCP 测试（Run + Ping）
└── README.md           # 项目文档
```

## 开发状态

当前版本 (v0.1.1) 实现了基本的传输层功能：

- [x] Transport 和 Connection 接口适配 MCP go-sdk（直接作为 `mcp.Transport` 使用）
- [x] MCP 消息到 MOQT 对象的映射（1 MCP message = 1 MOQT object/group）
- [x] 控制轨道（control tracks）的基本实现：
  - 命名空间：`["mcp", <session-id>, "control"]`
  - 轨道：`client-to-server` / `server-to-client`
- [x] 会话发现机制（`mcp/discovery` + `sessions` track，基于 FETCH）
- [x] 基于 QUIC + TLS 的端到端连通性测试（`TestMCPServerClient_RunAndPing`）
- [x] 基于 options 的传输构造（`WithAddr` / `WithTLSServerConfig` / `WithTLSClientConfig` / `WithQUICConfig`）
- [ ] 草案中描述的资源轨道（resources tracks）完整支持
- [ ] 工具轨道、提示轨道、通知轨道等专用 track 的拆分与流控策略
- [ ] 多会话 / 多命名空间场景下的 track 布局与路由
- [ ] 更细致的错误码映射与恢复策略（与 IETF 草案 error model 对齐）

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request。
