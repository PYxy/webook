package web

import (
	"gitee.com/geekbang/basic-go/webook/payment/service/wechat"
	"gitee.com/geekbang/basic-go/webook/pkg/ginx"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"net/http"
)

type WechatHandler struct {
	handler   *notify.Handler
	l         logger.LoggerV1
	nativeSvc *wechat.NativePaymentService
}

func NewWechatHandler(handler *notify.Handler,
	nativeSvc *wechat.NativePaymentService,
	l logger.LoggerV1) *WechatHandler {
	return &WechatHandler{
		handler:   handler,
		nativeSvc: nativeSvc,
		l:         l}
}

func (h *WechatHandler) RegisterRoutes(server *gin.Engine) {
	server.GET("/hello", func(context *gin.Context) {
		context.String(http.StatusOK, "我进来了")
	})
	server.Any("/pay/callback", ginx.Wrap(h.HandleNative))
}

func (h *WechatHandler) HandleNative(ctx *gin.Context) (ginx.Result, error) {
	transaction := &payments.Transaction{}
	// 第一个返回值里面的内容我们暂时用不上
	_, err := h.handler.ParseNotifyRequest(ctx, ctx.Request, transaction)
	if err != nil {
		return ginx.Result{}, err
	}
	err = h.nativeSvc.HandleCallback(ctx, transaction)
	return ginx.Result{}, err
}
