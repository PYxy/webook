package grpc

import (
	"context"
	"fmt"
)

type Server struct {
	UnimplementedUserServiceServer
	Name string
	Port string
}

var _ UserServiceServer = &Server{}

func (s *Server) GetById(ctx context.Context, request *GetByIdRequest) (*GetByIdResponse, error) {
	fmt.Println(s.Name)
	return &GetByIdResponse{
		User: &User{
			Id:   123,
			Name: fmt.Sprintf("%v:%s:123456", s.Name, s.Port),
		},
	}, nil
}

type Server2 struct {
	UnimplementedUserServiceServer
	Name string
	Port string
}

var _ UserServiceServer = &Server2{}

func (s *Server2) GetById(ctx context.Context, request *GetByIdRequest) (*GetByIdResponse, error) {

	return &GetByIdResponse{
		User: &User{
			Id:   123,
			Name: fmt.Sprintf("%v:%s:123456", s.Name, s.Port),
		},
	}, nil
}
