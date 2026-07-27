# grpc-demo

一个基于 Go 语言的 gRPC 示例项目，演示如何使用 Protocol Buffers 定义服务，并通过 **etcd 实现服务注册与发现**。

## 项目结构

```
grpc-demo/
├── proto/
│   ├── hello.proto          # Greeter 服务定义（SayHello、GetUserPoints）
│   ├── user.proto           # UserService 服务定义（CreateUser、GetUser）
│   ├── hello/               # protoc 生成的 Go 代码
│   └── user/                # protoc 生成的 Go 代码
├── server/
│   ├── main.go              # gRPC 服务端入口，包含服务注册到 etcd
│   └── handler/
│       └── user_handler.go  # UserService 业务逻辑实现
├── client/
│   └── main.go              # gRPC 客户端，自定义 etcd resolver 实现服务发现
├── go.mod
└── README.md
```

## 基本原理

### 1. Protocol Buffers 服务定义

gRPC 使用 `.proto` 文件描述服务接口和消息格式，通过 `protoc` 编译器自动生成各语言的桩代码。

本项目定义了两个服务：

- **Greeter**（`proto/hello.proto`）
  - `SayHello` — 一元 RPC，客户端发送姓名，服务端返回问候语
  - `GetUserPoints` — 一元 RPC，根据用户 ID 查询积分

- **UserService**（`proto/user.proto`）
  - `CreateUser` — 创建用户，返回分配的用户 ID
  - `GetUser` — 根据用户 ID 查询用户信息

### 2. 服务端实现

服务端核心流程：

1. **连接 etcd**：通过 `clientv3.New` 连接到 `localhost:2379` 的 etcd 实例
2. **创建租约**：`etcdClient.Grant(ctx, 10)` 创建 TTL 为 10 秒的租约，用于服务健康检查
3. **监听端口**：`net.Listen("tcp", "0.0.0.0:50051")` 在 50051 端口监听
4. **注册服务到 etcd**：
   - 使用 `endpoints.Endpoint` 结构体封装服务地址
   - 序列化为 JSON：`{"Addr":"10.186.40.167:50051","Metadata":null}`
   - 通过 `etcdClient.Put` 将 JSON 写入 etcd，key 为 `my-grpc-service/<IP:Port>`，绑定租约
5. **保持租约**：启动后台 goroutine 通过 `KeepAlive` 自动续租，防止 etcd 过期删除服务记录
6. **启动 gRPC 服务**：注册 Greeter 和 UserService 实现，阻塞监听请求

业务逻辑通过嵌入 `UnimplementedGreeterServer` / `UnimplementedUserServiceServer` 实现，保证向前兼容。数据存储使用内存 map 模拟。

### 3. 客户端服务发现（自定义 etcd Resolver）

由于 etcd v3.7+ 移除了官方的 resolver 包，本项目实现了一个**自定义的 gRPC Resolver**，需要处理服务器端存储的 JSON 格式数据：

#### etcd 存储格式
- **服务器端**：使用 `endpoints.Endpoint` 结构体，序列化为 JSON 存储在 etcd 中
  ```json
  {"Addr":"10.186.40.167:50051","Metadata":null}
  ```
- **key 格式**：`my-grpc-service/<IP:Port>`
- **租约绑定**：每个服务地址绑定 TTL 为 10 秒的租约，自动续租

#### 自定义 Resolver 实现
1. **自定义 Resolver Builder**：实现 `resolver.Builder` 接口，注册 scheme 为 `etcd`
2. **JSON 解析**：从 etcd 读取数据后，先尝试解析 JSON 格式，提取 `Addr` 字段的实际地址
   ```go
   var endpoint struct {
       Addr     string `json:"Addr"`
       Metadata any    `json:"Metadata"`
   }
   if err := json.Unmarshal(value, &endpoint); err == nil {
       addr = endpoint.Addr  // 使用解析后的地址
   } else {
       addr = value          // 兼容纯文本格式
   }
   ```
3. **服务发现**：通过 `etcdClient.Get(ctx, "my-grpc-service/", WithPrefix())` 查询所有服务地址
4. **轮询更新**：启动 5 秒间隔的定时器，周期性从 etcd 拉取最新地址列表，通过 `cc.UpdateState` 通知 gRPC
5. **建立连接**：`grpc.NewClient("etcd:///my-grpc-service")` 使用自定义 resolver 解析服务地址

### 4. 服务注册与发现流程

```
服务端                                    etcd                                    客户端
  │                                        │                                        │
  ├── 创建 Endpoint ────────────────────────┤                                        │
  ├── JSON 序列化 ─────────────────────────┤                                        │
  ├── Put("my-grpc-service/IP:Port",     ─→│                                        │
  │    {"Addr":"IP:Port","Metadata":null})  │                                        │
  ├── KeepAlive(租约续租) ──────────────────→│                                        │
  │                                        │    ←── Get("my-grpc-service/", prefix) ──┤
  │                                        │    ──→ 返回 JSON 地址列表 ─────────────────┤
  │                                        │                                        ├── JSON 解析提取 Addr
  │                                        │                                        ├── 建立 gRPC 连接
  │    ←─────── gRPC 请求 (SayHello) ──────┼─────────────────────────────────────────┤
  │    ──────── gRPC 响应 ────────────────→┼─────────────────────────────────────────┤
```

## 功能演示

| 功能 | 服务 | RPC 方法 | 说明 |
|------|------|----------|------|
| 简单问候 | Greeter | `SayHello` | 一元 RPC 基础示例 |
| 查询积分 | Greeter | `GetUserPoints` | 含业务逻辑（VIP 用户加分） |
| 创建用户 | UserService | `CreateUser` | 原子 ID 生成，模拟数据库存储 |
| 查询用户 | UserService | `GetUser` | 根据 ID 查询用户详情 |

## 前置条件

- Go 1.25+
- etcd 运行在 `localhost:2379`
- 依赖：
  - `google.golang.org/grpc v1.82.1`
  - `google.golang.org/protobuf v1.36.11`
  - `go.etcd.io/etcd/client/v3 v3.7.1`

## 运行方式

### 1. 确保 etcd 已启动

```bash
# 检查 etcd 是否运行
curl http://localhost:2379/health
# 期望输出: {"health":"true","reason":""}
```

### 2. 启动服务端

```bash
go run ./server/
```

服务端将在 `0.0.0.0:50051` 监听，并将服务地址注册到 etcd。

### 3. 启动客户端（另开终端）

```bash
# 使用默认名字 "World"
go run ./client/

# 自定义名字
go run ./client/ 张三
```

### 4. 预期输出

**服务端：**
```
✅ etcd 连接成功
🔑 租约 ID: xxxxxxxx
📍 服务地址: 10.186.40.167:50051
✅ 服务注册成功: my-grpc-service/10.186.40.167:50051 -> 10.186.40.167:50051
🚀 服务端已启动，监听端口 50051...
```

**客户端：**
```
✅ etcd 连接正常
✅ gRPC 连接创建成功
发现服务地址: 10.186.40.167:50051
✅ SayHello 响应: 你好, World
✅ GetUserPoints 响应: 用户 张三 的积分是 160
✅ CreateUser 响应: 用户 王五 创建成功 (用户ID: 1)
✅ GetUser 响应: 用户 王五, 年龄 28
```

## 技术栈

- **语言**：Go
- **RPC 框架**：gRPC + Protocol Buffers
- **服务注册与发现**：etcd（自定义 Resolver，支持 JSON 格式解析）
- **通信协议**：HTTP/2（gRPC 默认传输协议）
- **序列化**：Protocol Buffers（二进制高效序列化）
- **数据格式**：JSON（etcd 服务端存储）、gRPC（服务间通信）
