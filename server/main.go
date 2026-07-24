package main

import (
	"context"
	"log"
	"net"

	pb "grpc-demo/proto/hello" // 别名导入生成的代码
	userpb "grpc-demo/proto/user"
	"grpc-demo/server/handler"

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
	// 收到客户端请求中的 name，构造问候语
	return &pb.HelloReply{Message: "你好, " + req.Name}, nil
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
	// 1. 监听本地的 50051 端口
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}

	// 2. 创建 gRPC 服务器实例
	s := grpc.NewServer()

	// 3. 将我们的服务实现注册到 gRPC 服务器上
	pb.RegisterGreeterServer(s, &server{})
	// 3. 🆕 注册 UserService 服务（新增）
	userpb.RegisterUserServiceServer(s, handler.NewUserServiceHandler())

	log.Println("服务端已启动，监听端口 50051...")

	// 4. 开始接收并处理客户端请求
	if err := s.Serve(lis); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
