package middleware

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer 初始化 Jaeger 连接
// serviceName: 服务名，比如 "note-api"
// jaegerEndpoint: Jaeger 地址，比如 "http://localhost:14268/api/traces" (本地)
func InitTracer(serviceName, jaegerEndpoint string) (*tracesdk.TracerProvider, error) {
	// 1. 创建 Jaeger 导出器
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, err
	}

	// 2. 创建资源属性 (标识这是哪个服务)
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("environment", "development"),
		)),
	)

	// 3. 设置全局 Tracer
	otel.SetTracerProvider(tp)

	return tp, nil
}
