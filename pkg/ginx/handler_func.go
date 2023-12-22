package ginx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

// 受制于泛型，我们这里只能使用包变量，我深恶痛绝的包变量
var log logger.LoggerV1 = logger.NewNoOpLogger()

// 包变量导致我们这个地方的代码非常垃圾
var vector *prometheus.CounterVec

func InitCounter(opt prometheus.CounterOpts) {
	vector = prometheus.NewCounterVec(opt, []string{"code"})
	prometheus.MustRegister(vector)
}

func SetLogger(l logger.LoggerV1) {
	log = l
}

// WrapReq 。
func WrapReq[Req any](fn func(*gin.Context, Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			log.Error("解析请求失败", logger.Error(err))
			return
		}
		res, err := fn(ctx, req)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		ctx.JSON(http.StatusOK, res)
	}
}

func Wrap(fn func(*gin.Context) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := fn(ctx)
		if err != nil {
			log.Error("执行业务逻辑失败",
				logger.Error(err))
		}
		vector.WithLabelValues(strconv.Itoa(res.Code)).Inc()
		ctx.JSON(http.StatusOK, res)
	}
}
