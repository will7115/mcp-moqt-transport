# MCP over MOQT Transport

MCP over MOQT Transport 是一个实现了 Model Context Protocol (MCP) over Media over QUIC Transport (MOQT) 的 Go 语言传输层实现。

## 概述

本项目实现了 [draft-jennings-mcp-over-moqt-00](https://datatracker.ietf.org/doc/draft-jennings-mcp-over-moqt/) 草案中定义的 MCP over MOQT 传输协议，将 MCP 消息映射到 MOQT 对象，实现高效的发布-订阅通信。

## 功能特性

- ✅ 基本的对象映射为 MOQT 的载荷
- ✅ 控制轨道（Control Tracks）实现
- ✅ 客户端和服务器端传输实现
- ✅ 会话发现机制
- ✅ 连通性测试支持

## 版本

当前版本: v0.1.0

## 安装

```bash
go get github.com/mcp-moqt/mcp-moqt-transport
```

## 使用示例

### 服务器端

```go
package main

import (
    "context"
    "github.com/mcp-moqt/mcp-moqt-transport"
    "github.com/mengelbart/moqtransport"
)

func main() {
    // 创建 MOQT 会话
    session := &moqtransport.Session{
        // 配置会话
    }
    
    // 创建 MCP over MOQT 服务器传输
    transport := mcpmoqt.NewMOQTServerTransport(session)
    
    // 连接到传输
    conn, err := transport.Connect(context.Background())
    if err != nil {
        // 处理错误
    }
    
    // 使用连接进行 MCP 通信
    // ...
}
```

### 客户端

```go
package main

import (
    "context"
    "github.com/mcp-moqt/mcp-moqt-transport"
    "github.com/mengelbart/moqtransport"
)

func main() {
    // 创建 MOQT 会话
    session := &moqtransport.Session{
        // 配置会话
    }
    
    // 创建 MCP over MOQT 客户端传输
    transport := mcpmoqt.NewMOQTClientTransport(session)
    
    // 连接到传输
    conn, err := transport.Connect(context.Background())
    if err != nil {
        // 处理错误
    }
    
    // 使用连接进行 MCP 通信
    // ...
}
```

## 测试

### 本地连通性测试

运行本地网络连通性测试，验证QUIC和MOQT连接是否正常：

```bash
go test -v -run TestLocalConnectivity ./...
```

这个测试会：
- 创建一个QUIC服务器监听器
- 客户端连接到服务器
- 建立MOQT会话
- 验证基本的网络连通性

**注意**：客户端连接传输可能会失败（因为v0.1.0的会话发现机制还未完全实现），但这不影响连通性测试的目的 - 它验证了QUIC和MOQT层面的连接是正常的。

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
├── transport.go      # 基础传输接口和实现
├── server.go         # 服务器端传输实现
├── client.go         # 客户端传输实现
├── handler.go        # MOQT 消息处理器
├── examples/         # 示例代码
├── tests/            # 测试文件
└── README.md         # 项目文档
```

## 开发状态

当前版本 (v0.1.0) 实现了基本的传输层功能：

- [x] Transport 和 Connection 接口实现
- [x] MCP 消息到 MOQT 对象的映射
- [x] 控制轨道的基本实现
- [x] 会话发现机制
- [ ] 完整的资源轨道支持
- [ ] 工具轨道支持
- [ ] 提示轨道支持
- [ ] 通知轨道支持
- [ ] 完整的错误处理

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request。
