package main

import (
	"context"
	"log"
	"time"

	hellopb "grpc-demo/proto/hello"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("尝试连接 10.186.40.167:50051...")

	conn, err := grpc.DialContext(
		context.Background(),
		"10.186.40.167:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second),
	)
	if err != nil {
		log.Fatalf("❌ 直接连接失败: %v", err)
	}
	defer conn.Close()
	log.Println("✅ 直接连接成功")

	client := hellopb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.SayHello(ctx, &hellopb.HelloRequest{Name: "World"})
	if err != nil {
		log.Fatalf("❌ 调用失败: %v", err)
	}
	log.Printf("✅ 响应: %s", resp.Message)
}
