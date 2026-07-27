package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	hellopb "grpc-demo/proto/hello"
	userpb "grpc-demo/proto/user"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// ⭐ 自定义 resolver，读取纯字符串格式
type etcdResolverBuilder struct {
	client *clientv3.Client
}

func (b *etcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &etcdResolver{
		client: b.client,
		cc:     cc,
		stopCh: make(chan struct{}),
	}
	r.resolve()
	go r.watch()
	return r, nil
}

func (b *etcdResolverBuilder) Scheme() string {
	return "etcd"
}

type etcdResolver struct {
	client *clientv3.Client
	cc     resolver.ClientConn
	stopCh chan struct{}
}

func (r *etcdResolver) resolve() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 读取所有以 my-grpc-service 开头的 key
	resp, err := r.client.Get(ctx, "my-grpc-service/", clientv3.WithPrefix())
	if err != nil {
		log.Printf("从 etcd 读取服务地址失败: %v", err)
		return
	}

	var addrs []resolver.Address
	for _, kv := range resp.Kvs {
		value := string(kv.Value)

		// 尝试解析 JSON 格式的 Endpoint
		var endpoint struct {
			Addr     string `json:"Addr"`
			Metadata any    `json:"Metadata"`
		}

		if err := json.Unmarshal([]byte(value), &endpoint); err == nil {
			// 解析成功，使用 Addr 字段
			addrs = append(addrs, resolver.Address{Addr: endpoint.Addr})
			log.Printf("发现服务地址: %s", endpoint.Addr)
		} else {
			// 解析失败，可能是纯字符串格式（兼容旧格式）
			addrs = append(addrs, resolver.Address{Addr: value})
			log.Printf("发现服务地址: %s", value)
		}
	}

	if len(addrs) == 0 {
		log.Println("⚠️ etcd 中未发现任何服务地址")
	} else {
		r.cc.UpdateState(resolver.State{Addresses: addrs})
	}
}

func (r *etcdResolver) watch() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.resolve()
		case <-r.stopCh:
			return
		}
	}
}

func (r *etcdResolver) ResolveNow(options resolver.ResolveNowOptions) {
	r.resolve()
}

func (r *etcdResolver) Close() {
	close(r.stopCh)
}

// func main() {
// 	// 1. 创建 etcd 客户端
// 	etcdClient, err := clientv3.New(clientv3.Config{
// 		Endpoints:   []string{"localhost:2379"},
// 		DialTimeout: 5 * time.Second,
// 	})
// 	if err != nil {
// 		log.Fatalf("连接 etcd 失败: %v", err)
// 	}
// 	defer etcdClient.Close()

// 	// 验证 etcd 连通性
// 	ctx, checkCancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	_, err = etcdClient.Get(ctx, "health-check")
// 	checkCancel()
// 	if err != nil {
// 		log.Fatalf("etcd 连通性检查失败: %v", err)
// 	}
// 	log.Println("✅ etcd 连接正常")

// 	// 2. ⭐ 注册自定义 resolver（不使用 etcdresolver 包）
// 	builder := &etcdResolverBuilder{client: etcdClient}
// 	resolver.Register(builder)

// 	// 3. 建立 gRPC 连接（不阻塞等待）
// 	conn, err := grpc.NewClient(
// 		"etcd:///my-grpc-service",
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 	)
// 	if err != nil {
// 		log.Fatalf("连接服务失败: %v", err)
// 	}
// 	defer conn.Close()
// 	log.Println("✅ gRPC 连接创建成功")

// 	// 4. 等待一小段时间让连接建立和解析完成
// 	time.Sleep(1 * time.Second)

// 	// 5. 创建客户端
// 	helloClient := hellopb.NewGreeterClient(conn)
// 	userClient := userpb.NewUserServiceClient(conn)

// 	// 6. 准备请求数据
// 	name := "World"
// 	if len(os.Args) > 1 {
// 		name = os.Args[1]
// 	}

// 	// 7. 调用 SayHello（超时设长一点）
// 	callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer callCancel()

// 	helloResp, err := helloClient.SayHello(callCtx, &hellopb.HelloRequest{Name: name})
// 	if err != nil {
// 		log.Fatalf("调用失败: %v", err)
// 	}
// 	log.Printf("✅ SayHello 响应: %s", helloResp.Message)

// 	// 8. 调用 GetUserPoints
// 	pointsResp, err := helloClient.GetUserPoints(context.Background(), &hellopb.UserRequest{UserId: 1})
// 	if err != nil {
// 		log.Fatalf("GetUserPoints 调用失败: %v", err)
// 	}
// 	log.Printf("✅ GetUserPoints 响应: 用户 %s 的积分是 %d", pointsResp.UserName, pointsResp.Points)

// 	// 9. 调用 CreateUser
// 	createResp, err := userClient.CreateUser(context.Background(), &userpb.CreateUserRequest{
// 		Name: "王五",
// 		Age:  28,
// 	})
// 	if err != nil {
// 		log.Fatalf("CreateUser 调用失败: %v", err)
// 	}
// 	log.Printf("✅ CreateUser 响应: %s (用户ID: %d)", createResp.Message, createResp.UserId)

// 	// 10. 调用 GetUser
// 	getResp, err := userClient.GetUser(context.Background(), &userpb.GetUserRequest{UserId: 1})
// 	if err != nil {
// 		log.Fatalf("GetUser 调用失败: %v", err)
// 	}
// 	log.Printf("✅ GetUser 响应: 用户 %s, 年龄 %d", getResp.Name, getResp.Age)
// }

func main() {
	// 1. 创建 etcd 客户端
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("连接 etcd 失败: %v", err)
	}
	defer etcdClient.Close()

	// 验证 etcd 连通性
	ctx, checkCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = etcdClient.Get(ctx, "health-check")
	checkCancel()
	if err != nil {
		log.Fatalf("etcd 连通性检查失败: %v", err)
	}
	log.Println("✅ etcd 连接正常")

	// 2. ⭐ 注册自定义 resolver
	builder := &etcdResolverBuilder{client: etcdClient}
	resolver.Register(builder)

	// 3. 建立 gRPC 连接
	conn, err := grpc.NewClient(
		"etcd:///my-grpc-service",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("连接服务失败: %v", err)
	}
	defer conn.Close()
	log.Println("✅ gRPC 连接创建成功")

	// 4. 等待连接建立
	time.Sleep(1 * time.Second)

	// 5. 创建客户端
	helloClient := hellopb.NewGreeterClient(conn)
	userClient := userpb.NewUserServiceClient(conn)

	// 6. 准备请求数据
	name := "World"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	// 7. 调用 SayHello
	callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer callCancel()

	helloResp, err := helloClient.SayHello(callCtx, &hellopb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("调用失败: %v", err)
	}
	log.Printf("✅ SayHello 响应: %s", helloResp.Message)

	// 8. 调用 GetUserPoints
	pointsResp, err := helloClient.GetUserPoints(context.Background(), &hellopb.UserRequest{UserId: 1})
	if err != nil {
		log.Fatalf("GetUserPoints 调用失败: %v", err)
	}
	log.Printf("✅ GetUserPoints 响应: 用户 %s 的积分是 %d", pointsResp.UserName, pointsResp.Points)

	// 9. 调用 CreateUser
	createResp, err := userClient.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Name: "王五",
		Age:  28,
	})
	if err != nil {
		log.Fatalf("CreateUser 调用失败: %v", err)
	}
	log.Printf("✅ CreateUser 响应: %s (用户ID: %d)", createResp.Message, createResp.UserId)

	// 10. 调用 GetUser
	getResp, err := userClient.GetUser(context.Background(), &userpb.GetUserRequest{UserId: 1})
	if err != nil {
		log.Fatalf("GetUser 调用失败: %v", err)
	}
	log.Printf("✅ GetUser 响应: 用户 %s, 年龄 %d", getResp.Name, getResp.Age)
}
