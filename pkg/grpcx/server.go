package grpcx

import (
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	*grpc.Server
	Addr string
}

func (s *Server) Serve() error {
	//l, err := net.Listen("tcp", ":8090")
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	// 这边会阻塞，类似与 gin.Run
	return s.Server.Serve(l)
}
