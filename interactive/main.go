package main

import (
	intrv1 "gitee.com/geekbang/basic-go/webook/api/proto/gen/intr/v1"
	"gitee.com/geekbang/basic-go/webook/interactive/grpc"
	grpc2 "google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	server := grpc2.NewServer()
	// 这里暂时随便搞一下
	intrSvc := &grpc.InteractiveServiceServer{}
	intrv1.RegisterInteractiveServiceServer(server, intrSvc)
	// 监听 8090 端口，你可以随便写
	l, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}
	// 这边会阻塞，类似与 gin.Run
	err = server.Serve(l)
	log.Println(err)
}
