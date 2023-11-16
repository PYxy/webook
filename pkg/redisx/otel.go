package redisx

import (
	"context"
	"net"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TracingHook struct {
	tracer trace.Tracer
}

func (t TracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// 相当于，你这里啥也不干
		return next(ctx, network, addr)
	}

}

func (t TracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	//如果是redis集群用FulIName
	// span.SetAttributes(attribute.String("cmd.FulIName"，cmd.FulIName()))span.SetAttributes(attribute.String("cmd,Name", cmd,Name()))span.SetAttributes(attribute,String("cmd,string", cmd.string()))err := next(ctx, cmd)if err != nil fspan.RecordError(err)span.Setstatus(codes,Error, err.Error())

	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, span := t.tracer.Start(ctx, "redisx: "+cmd.Name(), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		//如果是redis集群用FulIName
		// span.SetAttributes(attribute.String("cmd.FulIName",cmd.FulIName()))
		span.SetAttributes(attribute.String("cmd.Name", cmd.Name()))
		span.SetAttributes(attribute.String("cmd.string", cmd.String()))
		err := next(ctx, cmd)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}
func (t TracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	//TODO implement me
	panic("implement me")
}
func NewTracingHook() *TracingHook {
	return &TracingHook{
		tracer: otel.GetTracerProvider().Tracer("webook/pkg/redisx/trace"),
	}
}

func Use(client *redis.Client) {
	client.AddHook(NewTracingHook())
}
