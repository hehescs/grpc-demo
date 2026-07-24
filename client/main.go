package main

import (
	"context"
	"log"
	"os"
	"time"

	hellopb "grpc-demo/proto/hello"
	userpb "grpc-demo/proto/user"

	"google.golang.org/grpc"
)

func main() {
	// 1. 连接到服务端（这里不使用 TLS 加密）
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	// 2. 创建客户端
	helloClient := hellopb.NewGreeterClient(conn)
	userClient := userpb.NewUserServiceClient(conn)

	// 3. 准备请求数据，名字可以从命令行参数获取，默认用 "World"
	name := "World"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	// 4. 发起 RPC 调用，设置超时时间为 1 秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	helloResp, err := helloClient.SayHello(ctx, &hellopb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("调用失败: %v", err)
	}

	// 5. 打印服务端的响应
	log.Printf("SayHello 响应: %s", helloResp.Message)

	// 🆕 调用新的 GetUserPoints
	pointsResp, err := helloClient.GetUserPoints(context.Background(), &hellopb.UserRequest{UserId: 1})
	if err != nil {
		log.Fatalf("GetUserPoints 调用失败: %v", err)
	}
	log.Printf("GetUserPoints 响应: 用户 %s 的积分是 %d", pointsResp.UserName, pointsResp.Points)

	// 调用新服务：创建用户
	createResp, _ := userClient.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Name: "王五",
		Age:  28,
	})
	log.Printf("CreateUser 响应: %s (用户ID: %d)", createResp.Message, createResp.UserId)

	// 调用新服务：查询用户
	getResp, _ := userClient.GetUser(context.Background(), &userpb.GetUserRequest{UserId: 1})
	log.Printf("GetUser 响应: 用户 %s, 年龄 %d", getResp.Name, getResp.Age)

}
