package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/gin-gonic/gin"
	"github.com/wordflowlab/agentsdk"
)

// TracingConfig 追踪配置
type TracingConfig struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string

	// OTLP Exporter 配置
	OTLPEndpoint string // e.g., "localhost:4318"
	OTLPInsecure bool

	// Sampling
	SamplingRate float64 // 0.0 - 1.0
}

// TracingManager 追踪管理器
type TracingManager struct {
	config   TracingConfig
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracingManager 创建追踪管理器
func NewTracingManager(config TracingConfig) (*TracingManager, error) {
	if !config.Enabled {
		return &TracingManager{config: config}, nil
	}

	// 设置默认值
	if config.ServiceName == "" {
		config.ServiceName = "agentsdk"
	}
	if config.ServiceVersion == "" {
		config.ServiceVersion = agentsdk.Version
	}
	if config.Environment == "" {
		config.Environment = "development"
	}
	if config.OTLPEndpoint == "" {
		config.OTLPEndpoint = "localhost:4318"
	}
	if config.SamplingRate == 0 {
		config.SamplingRate = 1.0 // 默认全采样
	}

	// 创建 OTLP HTTP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
	}
	if config.OTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建 Resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建 TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(provider)

	// 设置全局 Propagator (用于跨服务传播 trace context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	return &TracingManager{
		config:   config,
		provider: provider,
		tracer:   tracer,
	}, nil
}

// Middleware 返回 Gin 追踪中间件
func (t *TracingManager) Middleware() gin.HandlerFunc {
	if !t.config.Enabled || t.provider == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return otelgin.Middleware(t.config.ServiceName)
}

// Tracer 返回 tracer 实例
func (t *TracingManager) Tracer() trace.Tracer {
	return t.tracer
}

// Shutdown 关闭追踪提供者
func (t *TracingManager) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// StartSpan 开始一个新的 span
func (t *TracingManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, opts...)
}

// AddEvent 添加事件到当前 span
func AddEvent(ctx context.Context, name string, attributes ...interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		span.AddEvent(name)
	}
}

// SetAttribute 设置属性到当前 span
func SetAttribute(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		// 这里可以根据类型添加不同的属性
		// 简化处理，实际使用时应该根据 value 类型调用不同的方法
	}
}

// RecordError 记录错误到当前 span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() && err != nil {
		span.RecordError(err)
	}
}

// GetTraceID 获取当前 trace ID
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 获取当前 span ID
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
