package handler

import (
	"context"
	"fmt"
	"sync/atomic"

	pb "grpc-demo/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserServiceHandler 实现 UserServiceServer 接口
type UserServiceHandler struct {
	pb.UnimplementedUserServiceServer
	// 模拟数据库：用 map 存储用户数据，用 atomic 生成 ID
	users  map[int32]*pb.GetUserResponse
	nextID int32
}

// NewUserServiceHandler 构造函数
func NewUserServiceHandler() *UserServiceHandler {
	return &UserServiceHandler{
		users:  make(map[int32]*pb.GetUserResponse),
		nextID: 1,
	}
}

// CreateUser 实现创建用户
func (h *UserServiceHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// 1. 生成新用户 ID
	newID := atomic.AddInt32(&h.nextID, 1) - 1

	// 2. 保存到模拟数据库
	h.users[newID] = &pb.GetUserResponse{
		UserId: newID,
		Name:   req.Name,
		Age:    req.Age,
	}

	// 3. 返回响应
	return &pb.CreateUserResponse{
		UserId:  newID,
		Message: fmt.Sprintf("用户 %s 创建成功", req.Name),
	}, nil
}

// GetUser 实现查询用户
func (h *UserServiceHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// 1. 查询用户
	user, exists := h.users[req.UserId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "用户 %d 不存在", req.UserId)
	}

	// 2. 返回用户信息
	return user, nil
}
