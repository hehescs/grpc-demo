# grpc-demo

一个基于 Go 语言的 gRPC 入门示例项目，演示如何使用 Protocol Buffers 定义服务、生成代码，并实现服务端与客户端通信。

## 项目结构

```
grpc-demo/
├── proto/
│   ├── hello.proto          # Greeter 服务定义（SayHello、GetUserPoints）
│   ├── user.proto           # UserService 服务定义（CreateUser、GetUser）
│   ├── hello/               # protoc 生成的 Go 代码
│   └── user/                # protoc 生成的 Go 代码
├── server/
│   ├── main.go              # gRPC 服务端入口
│   └── handler/
│       └── user_handler.go  # UserService 业务逻辑实现
├── client/
│   └── main.go              # gRPC 客户端，调用所有服务
├── go.mod
└── README.md
```

## 基本原理

### 1. Protocol Buffers 服务定义

gRPC 使用 `.proto` 文件描述服务接口和消息格式，再通过 `protoc` 编译器自动生成各语言的桩代码。

本项目定义了两个服务：

- **Greeter**（`proto/hello.proto`）
  - `SayHello` — 一元 RPC，客户端发送姓名，服务端返回问候语
  - `GetUserPoints` — 一元 RPC，根据用户 ID 查询积分

- **UserService**（`proto/user.proto`）
  - `CreateUser` — 创建用户，返回分配的用户 ID
  - `GetUser` — 根据用户 ID 查询用户信息

### 2. 服务端实现

服务端核心流程：

1. **监听端口**：`net.Listen("tcp", ":50051")` 在本地 50051 端口监听
2. **创建 gRPC 服务器**：`grpc.NewServer()` 创建服务器实例
3. **注册服务**：通过 `RegisterGreeterServer` 和 `RegisterUserServiceServer` 将业务实现注册到 gRPC 服务器
4. **启动服务**：`s.Serve(lis)` 阻塞监听，接收并处理客户端请求

业务逻辑通过嵌入 `UnimplementedGreeterServer` / `UnimplementedUserServiceServer` 结构体实现，保证向前兼容。数据存储使用内存 map 模拟数据库。

### 3. 客户端调用

客户端核心流程：

1. **建立连接**：`grpc.NewClient("localhost:50051", grpc.WithInsecure())` 创建与服务端的连接（未使用 TLS）
2. **创建客户端桩**：通过 `NewGreeterClient` / `NewUserServiceClient` 生成类型安全的客户端实例
3. **发起 RPC 调用**：调用客户端方法，传入 `context.Context` 控制超时，gRPC 自动完成序列化、网络传输、反序列化
4. **处理响应**：接收结构化的响应对象，处理业务错误（如用户不存在时返回 `codes.NotFound`）

## 功能演示

| 功能 | 服务 | RPC 方法 | 说明 |
|------|------|----------|------|
| 简单问候 | Greeter | `SayHello` | 一元 RPC 基础示例 |
| 查询积分 | Greeter | `GetUserPoints` | 含业务逻辑（VIP 用户加分） |
| 创建用户 | UserService | `CreateUser` | 原子 ID 生成，模拟数据库存储 |
| 查询用户 | UserService | `GetUser` | 根据 ID 查询用户详情 |

## 运行方式

### 前置条件

- Go 1.25+
- 依赖：`google.golang.org/grpc v1.82.1`、`google.golang.org/protobuf v1.36.11`

### 启动服务端

```bash
go run server/main.go server/handler/user_handler.go
```

服务端将在 `localhost:50051` 监听。

### 启动客户端

```bash
# 使用默认名字 "World"
go run client/main.go

# 自定义名字
go run client/main.go 张三
```

### 预期输出

```
SayHello 响应: 你好, World
GetUserPoints 响应: 用户 张三 的积分是 160
CreateUser 响应: 用户 王五 创建成功 (用户ID: 1)
GetUser 响应: 用户 王五, 年龄 28
```

## 技术栈

- **语言**：Go
- **框架**：gRPC + Protocol Buffers
- **通信协议**：HTTP/2（gRPC 默认传输协议）
- **序列化**：Protocol Buffers（二进制高效序列化）
