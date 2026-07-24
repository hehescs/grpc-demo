package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "grpc-demo/proto/hello" // 别名导入生成的代码
	userpb "grpc-demo/proto/user"
	"grpc-demo/server/handler"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 模拟数据库（实际项目会连 MySQL/Redis）
var userDB = map[int32]struct {
	name   string
	points int32
}{
	1: {name: "张三", points: 150},
	2: {name: "李四", points: 320},
}

// server 结构体，用于实现 Greeter 服务接口
type server struct {
	pb.UnimplementedGreeterServer // 为未来兼容性而嵌入
}

// 实现 SayHello 方法，这是真正的业务逻辑
func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {

	log.Printf("收到请求: %s", req.Name)
	// 收到客户端请求中的 name，构造问候语
	return &pb.HelloReply{Message: "你好, " + req.Name}, nil

	// // ⭐ 故意制造一个 panic（空指针引用）
	// var nilMap map[string]string
	// nilMap["key"] = "value" // 这行会触发 panic

	// return &pb.HelloReply{Message: "你好, " + req.Name}, nil

}

// 🆕 新增：实现 GetUserPoints 方法（这才是真正的新业务逻辑）
func (s *server) GetUserPoints(ctx context.Context, req *pb.UserRequest) (*pb.PointsReply, error) {
	// 1. 从请求里拿到 user_id
	uid := req.UserId

	// 2. 查询"数据库"（这里用 map 模拟）
	user, exists := userDB[uid]
	if !exists {
		// 业务异常：用户不存在
		return nil, status.Errorf(codes.NotFound, "用户 %d 不存在", uid)
	}

	// 3. 业务逻辑：假设 VIP 用户（ID=1）额外加 10 分
	finalPoints := user.points
	if uid == 1 {
		finalPoints += 10
	}

	// 4. 构造响应返回
	return &pb.PointsReply{
		Points:   finalPoints,
		UserName: user.name,
	}, nil
}

func main() {
	// 1. 连接 etcd
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("连接 etcd 失败: %v", err)
	}
	defer etcdClient.Close()
	log.Println("✅ etcd 连接成功")

	// 2. 创建租约
	lease, err := etcdClient.Grant(context.TODO(), 10)
	if err != nil {
		log.Fatalf("创建租约失败: %v", err)
	}
	log.Printf("🔑 租约 ID: %x", lease.ID)

	// 3. 监听端口
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}

	// 4. 服务地址
	serverIP := "10.186.40.167"
	serverAddr := serverIP + ":50051"
	log.Printf("📍 服务地址: %s", serverAddr)

	// 5. 手动写入 etcd
	key := "my-grpc-service/" + serverAddr
	_, err = etcdClient.Put(
		context.TODO(),
		key,
		serverAddr, // 直接存地址
		clientv3.WithLease(lease.ID),
	)
	if err != nil {
		log.Fatalf("注册服务到 etcd 失败: %v", err)
	}
	log.Printf("✅ 服务注册成功: %s -> %s", key, serverAddr)

	// 6. 保持租约
	keepAliveChan, err := etcdClient.KeepAlive(context.TODO(), lease.ID)
	if err != nil {
		log.Fatalf("启动续租失败: %v", err)
	}
	go func() {
		for range keepAliveChan {
		}
		log.Println("⚠️ 租约已过期")
	}()

	// 7. 启动 gRPC 服务
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	userpb.RegisterUserServiceServer(s, handler.NewUserServiceHandler())

	log.Println("🚀 服务端已启动，监听端口 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
