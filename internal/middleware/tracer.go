package middleware

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation" // 【新增】引入 propagation
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func InitTracer(serviceName, jaegerEndpoint string) (*tracesdk.TracerProvider, error) {
	// 1. 创建 Jaeger 导出器
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, err
	}

	// 2. 创建 TracerProvider
	tp := tracesdk.NewTracerProvider(
		// 【优化】开发环境建议设置为 AlwaysSample，保证每条请求都被记录
		tracesdk.WithSampler(tracesdk.AlwaysSample()),

		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("environment", "development"),
		)),
	)

	// 3. 设置全局 Tracer
	otel.SetTracerProvider(tp)

	// 4. 设置全局传播器
	// 这行代码非常重要！它决定了 TraceID 如何在 HTTP Header (Traceparent) 中传递
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // 支持 W3C Trace Context 标准
		propagation.Baggage{},      // 支持携带额外信息
	))

	return tp, nil
}
