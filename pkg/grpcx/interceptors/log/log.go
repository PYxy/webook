package log

import (
	"context"
	"fmt"
	"gitee.com/geekbang/basic-go/webook/pkg/grpcx/interceptors"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime"
	"time"
)

type LoggerInterceptorBuilder struct {
	l logger.LoggerV1
	interceptors.Builder
}

func NewLoggerInterceptorBuilder(l logger.LoggerV1) *LoggerInterceptorBuilder {
	return &LoggerInterceptorBuilder{l: l}
}

func (b *LoggerInterceptorBuilder) BuildUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
		// 默认过滤掉该探活日志
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		var start = time.Now()
		var fields = make([]logger.Field, 0, 20)
		var event = "normal"

		defer func() {
			cost := time.Since(start)
			if rec := recover(); rec != nil {
				switch recType := rec.(type) {
				case error:
					err = recType
				default:
					err = fmt.Errorf("%v", rec)
				}
				stack := make([]byte, 4096)
				stack = stack[:runtime.Stack(stack, true)]
				event = "recover"
				err = status.New(codes.Internal, "panic, err "+err.Error()).Err()
			}
			st, _ := status.FromError(err)
			fields = append(fields,
				logger.String("type", "unary"),
				logger.String("code", st.Code().String()),
				logger.String("code_msg", st.Message()),
				logger.String("event", event),
				logger.String("method", info.FullMethod),
				logger.Int64("cost", cost.Milliseconds()),
				logger.String("peer", b.PeerName(ctx)),
				logger.String("peer_ip", b.PeerIP(ctx)),
			)
			b.l.Info("RPC调用", fields...)
		}()

		return handler(ctx, req)
	}
}
