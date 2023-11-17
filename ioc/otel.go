package ioc

import (
	"context"
	"fmt"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	trace2 "go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

func InitOTEL() func(ctx context.Context) {
	res, err := newResource("webook", "v0.0.1")
	if err != nil {
		panic(err)
	}
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	tp, err := newTraceProvider(res)
	if err != nil {
		panic(err)
	}
	//旧写法
	//otel.SetTracerProvider(tp)
	//return func(ctx context.Context) {
	//	tp.Shutdown(ctx)
	//}
	//动态调整开关
	newTp := &MyTracerProvider{
		Enable:        atomic.NewBool(true),
		SkipTraceList: atomic.NewString(viper.GetString("otel.skip")),
		nopProvider:   trace2.NewNoopTracerProvider(),
		provider:      tp,
	}

	otel.SetTracerProvider(newTp)

	return func(ctx context.Context) {
		tp.Shutdown(ctx)
	}
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

func newTraceProvider(res *resource.Resource) (*trace.TracerProvider, error) {
	exporter, err := zipkin.New(
		"http://156.236.71.5:9411/api/v2/spans")
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)
	return traceProvider, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

type MyTracerProvider struct {
	// 改原子操作
	Enable        *atomic.Bool
	SkipTraceList *atomic.String
	nopProvider   trace2.TracerProvider
	provider      trace2.TracerProvider
}
type MyTracerProviderv1 struct {
	// 改原子操作
	//Enable      *atomic.Bool
	//nopProvider trace2.TracerProvider
	//provider    trace2.TracerProvider
}

//func (m *MyTracerProvider) Tracer(name string, options ...trace2.TracerOption) trace2.Tracer {
//	if m.Enable.Load() {
//		return m.provider.Tracer(name, options...)
//	}
//	return m.nopProvider.Tracer(name, options...)
//}

func (m *MyTracerProvider) Tracer(name string, options ...trace2.TracerOption) trace2.Tracer {
	mtc := &MyTracer{
		Enable:        m.Enable,
		SkipTraceList: m.SkipTraceList,
		nopTracer:     m.nopProvider.Tracer(name, options...),
		tracer:        m.provider.Tracer(name, options...),
	}
	// 监听配置变更就可以了
	viper.OnConfigChange(func(in fsnotify.Event) {
		//只能知道变化了 但是 不知道那个数据发生变化了,只能重新读一次对应使用的配置
		//如
		status := viper.GetString("otel.status")
		fmt.Println("otel 状态发生变化:", status)
		mtc.Enable.Store(gocast.Bool(status))
		skip := viper.GetString("otel.skip")
		mtc.SkipTraceList.Store(skip)
		//fmt.Println(in.Name, in.Op)
	})

	return mtc
}

type MyTracer struct {
	Enable        *atomic.Bool
	SkipTraceList *atomic.String
	nopTracer     trace2.Tracer
	tracer        trace2.Tracer
}

func (m *MyTracer) Start(ctx context.Context, spanName string, opts ...trace2.SpanStartOption) (context.Context, trace2.Span) {
	if m.Enable.Load() {

		//不太行 这里的spanName 是 每次start 时候填的spanName
		//法1
		//除非strings.Contains(spanName,"ljy")  每个接口使用独立的关键字
		//法2
		//或者前端在header 中传一个 标志位 搞一个中间件来读取header 中的关键字放到ctx 中
		//在配置文件多加一个字段 针对开放的 spanName
		//atomic.String

		//skip := m.SkipTraceList.Load()
		//fmt.Println(spanName, skip)
		//skipSeps := strings.Split(skip, ",")
		//fmt.Println(skipSeps)
		//for _, val := range skipSeps {
		//	if strings.TrimSpace(val) == spanName {
		//		fmt.Println(spanName, "跳过")
		//		return m.nopTracer.Start(ctx, spanName, opts...)
		//	}
		//}
		return m.tracer.Start(ctx, spanName, opts...)
	}
	return m.nopTracer.Start(ctx, spanName, opts...)
}
