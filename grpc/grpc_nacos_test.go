package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NacosTestSuite struct {
	suite.Suite
	client naming_client.INamingClient
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (s *NacosTestSuite) SetupSuite() {
	//create clientConfig
	clientConfig := constant.ClientConfig{
		NamespaceId:         "e525eafa-f7d7-4029-83d9-008937f9d468",
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "debug",
	}
	// At least one ServerConfig
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      "localhost",
			ContextPath: "/nacos",
			Port:        8848,
			Scheme:      "http",
		},
	}
	cli, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	require.NoError(s.T(), err)
	s.client = cli
}

func (s *NacosTestSuite) TestClient() {
	rb := &nacosResolverBuilder{
		client: s.client,
	}
	cc, err := grpc.Dial("nacos:///user",
		grpc.WithResolvers(rb),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(s.T(), err)
	client := NewUserServiceClient(cc)
	resp, err := client.GetById(context.Background(), &GetByIdRequest{
		Id: 123,
	})
	require.NoError(s.T(), err)
	s.T().Log(resp.User)
}

func (s *NacosTestSuite) TestServer() {
	l, err := net.Listen("tcp", ":8090")
	require.NoError(s.T(), err)
	server := grpc.NewServer()
	RegisterUserServiceServer(server, &Server{})
	ok, err := s.client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          GetOutboundIP(),
		Port:        8090,
		ServiceName: "user",
		Enable:      true,
		Healthy:     true,
		Weight:      10,
	})
	require.NoError(s.T(), err)
	require.True(s.T(), ok)
	err = server.Serve(l)
	s.T().Log(err)
}

func TestNacos(t *testing.T) {
	suite.Run(t, new(NacosTestSuite))
}
