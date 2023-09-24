package logger

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"time"
)

type MiddlewareBuilder struct {
	allowReqBody  bool
	allowRespBody bool
	//logger        logger.LoggerV1  //这里要自己确认用什么日志级别
	loggerFunc func(ctx context.Context, al *AccessLog)
}

func NewBuilder(fn func(ctx context.Context, al *AccessLog)) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		loggerFunc: fn,
	}
}

func (b *MiddlewareBuilder) AllowReqBody() *MiddlewareBuilder {
	b.allowReqBody = true
	return b
}

func (b *MiddlewareBuilder) AllowRespBody() *MiddlewareBuilder {
	b.allowRespBody = true
	return b
}

func (b *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		url := ctx.Request.URL.String()
		if len(url) > 1024 {
			url = url[:1024]
		}
		al := &AccessLog{
			Meth: ctx.Request.Method,
			Url:  url,
		}
		if ctx.Request.Body != nil && b.allowReqBody {
			//body 读完就没有了 要写回去
			//body, _ := io.ReadAll(ctx.Request.Body)
			body, _ := ctx.GetRawData()
			//
			ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
			if len(body) > 1024 {
				body = body[:1024]
			}
			//这里很消耗资源
			al.ReqBody = string(body)

		}
		if b.allowRespBody {
			ctx.Writer = responeWrite{
				ResponseWriter: ctx.Writer,
				al:             al,
			}
		}
		defer func() {
			al.Duration = time.Since(start).String()
			//日志打印
			b.loggerFunc(ctx, al)
		}()
		//执行业务逻辑
		ctx.Next()

	}
}

type responeWrite struct {
	//组合 用于装饰部分方法
	gin.ResponseWriter
	al *AccessLog
}

func (r responeWrite) WriteHeader(statusCode int) {
	r.al.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r responeWrite) Write(data []byte) (int, error) {
	r.al.RespBody = string(data)
	return r.ResponseWriter.Write(data)
}

func (r responeWrite) WriteString(data string) (int, error) {
	//r.al.RespBody = data[:1024] 要注意长度 不然会panic
	r.al.RespBody = data
	return r.ResponseWriter.WriteString(data)
}

type AccessLog struct {
	//http 请求
	Meth string
	//url 整个请求的url
	Url string
	//请求体 可能很大
	ReqBody  string
	RespBody string
	Duration string
	Status   int
}
