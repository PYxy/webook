package grpc

import (
	"context"
	"fmt"
)

type Server struct {
	UnimplementedUserServiceServer
	Name string
}

var _ UserServiceServer = &Server{}

func (s *Server) GetById(ctx context.Context, request *GetByIdRequest) (*GetByIdResponse, error) {
	fmt.Println(s.Name)
	return &GetByIdResponse{
		User: &User{
			Id:   123,
			Name: fmt.Sprintf("%v:123456", s.Name),
		},
	}, nil
}
