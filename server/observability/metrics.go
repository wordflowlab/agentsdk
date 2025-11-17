package observability

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsManager Prometheus 指标管理器
type MetricsManager struct {
	// HTTP 指标
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec

	// 业务指标
	agentsTotal      prometheus.Gauge
	sessionsActive   prometheus.Gauge
	workflowsRunning prometheus.Gauge

	registry *prometheus.Registry
}

// NewMetricsManager 创建指标管理器
func NewMetricsManager(namespace string) *MetricsManager {
	if namespace == "" {
		namespace = "agentsdk"
	}

	m := &MetricsManager{
		registry: prometheus.NewRegistry(),
	}

	// HTTP 请求总数
	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求延迟
	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// 请求大小
	m.requestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// 响应大小
	m.responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Agents 总数
	m.agentsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "agents_total",
			Help:      "Total number of agents",
		},
	)

	// 活跃 Sessions
	m.sessionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sessions_active",
			Help:      "Number of active sessions",
		},
	)

	// 运行中的 Workflows
	m.workflowsRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workflows_running",
			Help:      "Number of running workflows",
		},
	)

	// 注册所有指标
	m.registry.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestSize,
		m.responseSize,
		m.agentsTotal,
		m.sessionsActive,
		m.workflowsRunning,
	)

	// 注册默认的 Go 和 Process 指标
	m.registry.MustRegister(prometheus.NewGoCollector())
	m.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return m
}

// Middleware Prometheus 中间件
func (m *MetricsManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 记录请求大小
		reqSize := computeRequestSize(c)
		m.requestSize.WithLabelValues(c.Request.Method, path).Observe(float64(reqSize))

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()

		m.requestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			statusClass(status),
		).Inc()

		m.requestDuration.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(duration)

		// 记录响应大小
		respSize := c.Writer.Size()
		if respSize > 0 {
			m.responseSize.WithLabelValues(c.Request.Method, path).Observe(float64(respSize))
		}
	}
}

// Handler Prometheus 指标暴露端点
func (m *MetricsManager) Handler() gin.HandlerFunc {
	h := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	return gin.WrapH(h)
}

// SetAgentsTotal 设置 Agents 总数
func (m *MetricsManager) SetAgentsTotal(count float64) {
	m.agentsTotal.Set(count)
}

// SetSessionsActive 设置活跃 Sessions 数量
func (m *MetricsManager) SetSessionsActive(count float64) {
	m.sessionsActive.Set(count)
}

// SetWorkflowsRunning 设置运行中的 Workflows 数量
func (m *MetricsManager) SetWorkflowsRunning(count float64) {
	m.workflowsRunning.Set(count)
}

// computeRequestSize 计算请求大小
func computeRequestSize(r *gin.Context) int {
	size := 0
	if r.Request.URL != nil {
		size += len(r.Request.URL.String())
	}
	size += len(r.Request.Method)
	size += len(r.Request.Proto)
	for name, values := range r.Request.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}
	if r.Request.ContentLength > 0 {
		size += int(r.Request.ContentLength)
	}
	return size
}

// statusClass 返回 HTTP 状态码类别
func statusClass(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "2xx"
	case status >= 300 && status < 400:
		return "3xx"
	case status >= 400 && status < 500:
		return "4xx"
	case status >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
