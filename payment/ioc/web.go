package ioc

import (
	"gitee.com/geekbang/basic-go/webook/payment/web"
	"gitee.com/geekbang/basic-go/webook/pkg/ginx"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

func InitGinServer(hdl *web.WechatHandler) *ginx.Server {
	engine := gin.Default()
	hdl.RegisterRoutes(engine)
	addr := viper.GetString("http.addr")
	ginx.InitCounter(prometheus.CounterOpts{
		Namespace: "daming_geektime",
		Subsystem: "webook_payment",
		Name:      "http",
	})
	return &ginx.Server{
		Engine: engine,
		Addr:   addr,
	}
}
