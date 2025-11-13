package telemetry

import (
	"context"
	"fmt"
	"time"
)

// Tracer 提供分布式追踪能力
// 参考 Google ADK-Go 的 OpenTelemetry 集成
type Tracer interface {
	// StartSpan 开始一个新的 span
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// Extract 从 carrier 中提取 trace context
	Extract(ctx context.Context, carrier interface{}) context.Context

	// Inject 将 trace context 注入到 carrier
	Inject(ctx context.Context, carrier interface{}) error
}

// Span 表示一个追踪片段
type Span interface {
	// End 结束 span
	End()

	// SetAttributes 设置属性
	SetAttributes(attrs ...Attribute)

	// SetStatus 设置状态
	SetStatus(code StatusCode, description string)

	// AddEvent 添加事件
	AddEvent(name string, attrs ...Attribute)

	// RecordError 记录错误
	RecordError(err error)

	// SpanContext 返回 span context
	SpanContext() SpanContext
}

// SpanContext 表示 span 的上下文信息
type SpanContext interface {
	TraceID() string
	SpanID() string
	IsSampled() bool
}

// Attribute 表示 span 的属性
type Attribute struct {
	Key   string
	Value interface{}
}

// StatusCode 表示 span 的状态码
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// SpanOption 配置 span 的选项
type SpanOption func(*SpanConfig)

// SpanConfig span 配置
type SpanConfig struct {
	Kind       SpanKind
	Attributes []Attribute
	StartTime  time.Time
}

// SpanKind span 类型
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// WithSpanKind 设置 span 类型
func WithSpanKind(kind SpanKind) SpanOption {
	return func(c *SpanConfig) {
		c.Kind = kind
	}
}

// WithAttributes 设置初始属性
func WithAttributes(attrs ...Attribute) SpanOption {
	return func(c *SpanConfig) {
		c.Attributes = append(c.Attributes, attrs...)
	}
}

// WithTimestamp 设置开始时间
func WithTimestamp(t time.Time) SpanOption {
	return func(c *SpanConfig) {
		c.StartTime = t
	}
}

// 常用属性构造函数
func String(key, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

func Int(key string, value int) Attribute {
	return Attribute{Key: key, Value: value}
}

func Int64(key string, value int64) Attribute {
	return Attribute{Key: key, Value: value}
}

func Float64(key string, value float64) Attribute {
	return Attribute{Key: key, Value: value}
}

func Bool(key string, value bool) Attribute {
	return Attribute{Key: key, Value: value}
}

// NoopTracer 空实现的 tracer，用于禁用追踪
type NoopTracer struct{}

func (t *NoopTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &NoopSpan{}
}

func (t *NoopTracer) Extract(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *NoopTracer) Inject(ctx context.Context, carrier interface{}) error {
	return nil
}

// NoopSpan 空实现的 span
type NoopSpan struct{}

func (s *NoopSpan) End()                                              {}
func (s *NoopSpan) SetAttributes(attrs ...Attribute)                  {}
func (s *NoopSpan) SetStatus(code StatusCode, description string)     {}
func (s *NoopSpan) AddEvent(name string, attrs ...Attribute)          {}
func (s *NoopSpan) RecordError(err error)                             {}
func (s *NoopSpan) SpanContext() SpanContext                          { return &NoopSpanContext{} }

// NoopSpanContext 空实现的 span context
type NoopSpanContext struct{}

func (c *NoopSpanContext) TraceID() string   { return "" }
func (c *NoopSpanContext) SpanID() string    { return "" }
func (c *NoopSpanContext) IsSampled() bool   { return false }

// SimpleTracer 简单的内存 tracer 实现
// 用于开发和测试环境
type SimpleTracer struct {
	spans []*SimpleSpan
}

func NewSimpleTracer() *SimpleTracer {
	return &SimpleTracer{
		spans: make([]*SimpleSpan, 0),
	}
}

func (t *SimpleTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	config := &SpanConfig{
		Kind:       SpanKindInternal,
		Attributes: make([]Attribute, 0),
		StartTime:  time.Now(),
	}

	for _, opt := range opts {
		opt(config)
	}

	span := &SimpleSpan{
		name:       name,
		startTime:  config.StartTime,
		attributes: config.Attributes,
		events:     make([]SpanEvent, 0),
		tracer:     t,
	}

	t.spans = append(t.spans, span)
	return ctx, span
}

func (t *SimpleTracer) Extract(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *SimpleTracer) Inject(ctx context.Context, carrier interface{}) error {
	return nil
}

// GetSpans 获取所有 span (用于测试)
func (t *SimpleTracer) GetSpans() []*SimpleSpan {
	return t.spans
}

// SimpleSpan 简单的 span 实现
type SimpleSpan struct {
	name        string
	startTime   time.Time
	endTime     time.Time
	attributes  []Attribute
	events      []SpanEvent
	status      StatusCode
	statusDesc  string
	err         error
	tracer      *SimpleTracer
}

type SpanEvent struct {
	Name       string
	Timestamp  time.Time
	Attributes []Attribute
}

func (s *SimpleSpan) End() {
	s.endTime = time.Now()
}

func (s *SimpleSpan) SetAttributes(attrs ...Attribute) {
	s.attributes = append(s.attributes, attrs...)
}

func (s *SimpleSpan) SetStatus(code StatusCode, description string) {
	s.status = code
	s.statusDesc = description
}

func (s *SimpleSpan) AddEvent(name string, attrs ...Attribute) {
	s.events = append(s.events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

func (s *SimpleSpan) RecordError(err error) {
	s.err = err
	s.status = StatusCodeError
}

func (s *SimpleSpan) SpanContext() SpanContext {
	return &SimpleSpanContext{
		traceID: fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		spanID:  fmt.Sprintf("span-%d", time.Now().UnixNano()),
	}
}

// 获取 span 信息 (用于测试和调试)
func (s *SimpleSpan) Name() string              { return s.name }
func (s *SimpleSpan) StartTime() time.Time      { return s.startTime }
func (s *SimpleSpan) EndTime() time.Time        { return s.endTime }
func (s *SimpleSpan) Duration() time.Duration   { return s.endTime.Sub(s.startTime) }
func (s *SimpleSpan) Attributes() []Attribute   { return s.attributes }
func (s *SimpleSpan) Events() []SpanEvent       { return s.events }
func (s *SimpleSpan) Status() StatusCode        { return s.status }
func (s *SimpleSpan) StatusDescription() string { return s.statusDesc }
func (s *SimpleSpan) Error() error              { return s.err }

// SimpleSpanContext 简单的 span context 实现
type SimpleSpanContext struct {
	traceID string
	spanID  string
}

func (c *SimpleSpanContext) TraceID() string { return c.traceID }
func (c *SimpleSpanContext) SpanID() string  { return c.spanID }
func (c *SimpleSpanContext) IsSampled() bool { return true }

// 全局默认 tracer
var globalTracer Tracer = &NoopTracer{}

// SetGlobalTracer 设置全局 tracer
func SetGlobalTracer(tracer Tracer) {
	globalTracer = tracer
}

// GetGlobalTracer 获取全局 tracer
func GetGlobalTracer() Tracer {
	return globalTracer
}

// StartSpan 使用全局 tracer 开始 span
func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return globalTracer.StartSpan(ctx, name, opts...)
}
