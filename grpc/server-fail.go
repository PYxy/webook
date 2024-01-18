package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerFail struct {
	UnimplementedUserServiceServer
	Name string
	Port string
}

var _ UserServiceServer = &ServerFail{}

func (s *ServerFail) GetById(ctx context.Context, request *GetByIdRequest) (*GetByIdResponse, error) {
	fmt.Println("ServerFail 测试")
	return &GetByIdResponse{
		User: &User{
			Id:   123,
			Name: fmt.Sprintf("%v:%s:123456", s.Name, s.Port),
		},
	}, status.Error(codes.Unavailable, "不可用")
}
