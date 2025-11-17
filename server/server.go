package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wordflowlab/agentsdk"
	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/server/auth"
	"github.com/wordflowlab/agentsdk/server/observability"
	"github.com/wordflowlab/agentsdk/server/ratelimit"
)

// Server represents the AgentSDK production server
type Server struct {
	config *Config
	router *gin.Engine
	server *http.Server
	store  store.Store

	// Dependencies (will be injected)
	deps *Dependencies

	// Auth & Observability
	authManager   *auth.Manager
	rbac          *auth.RBAC
	metrics       *observability.MetricsManager
	healthChecker *observability.HealthChecker
	tracing       *observability.TracingManager
	rateLimiter   ratelimit.Limiter
}

// Dependencies holds all dependencies for the server
type Dependencies struct {
	Store     store.Store
	AgentDeps *agent.Dependencies
}

// New creates a new Server instance with the given configuration
func New(config *Config, deps *Dependencies, opts ...Option) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	// Set Gin mode based on config
	if config.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s := &Server{
		config: config,
		router: gin.New(),
		store:  deps.Store,
		deps:   deps,
	}

	// Initialize auth and observability
	s.initializeAuthAndObservability()

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// initializeAuthAndObservability initializes authentication and observability components
func (s *Server) initializeAuthAndObservability() {
	// Initialize Auth Manager
	if s.config.Auth.APIKey.Enabled || s.config.Auth.JWT.Enabled {
		s.authManager = auth.NewManager(auth.AuthMethodAPIKey)

		// Register API Key authenticator
		if s.config.Auth.APIKey.Enabled {
			apiKeyStore := auth.NewMemoryAPIKeyStore()
			apiKeyAuth := auth.NewAPIKeyAuthenticator(apiKeyStore)
			s.authManager.Register(apiKeyAuth)
		}

		// Register JWT authenticator
		if s.config.Auth.JWT.Enabled {
			jwtAuth := auth.NewJWTAuthenticator(auth.JWTConfig{
				SecretKey:      s.config.Auth.JWT.Secret,
				Issuer:         "agentsdk",
				ExpiryDuration: time.Duration(s.config.Auth.JWT.Expiry) * time.Second,
			})
			s.authManager.Register(jwtAuth)
		}

		// Initialize RBAC
		s.rbac = auth.NewRBAC()
	}

	// Initialize Metrics
	if s.config.Observability.Metrics.Enabled {
		s.metrics = observability.NewMetricsManager("agentsdk")
	}

	// Initialize Health Checker
	if s.config.Observability.HealthCheck.Enabled {
		s.healthChecker = observability.NewHealthChecker(agentsdk.Version)

		// Register store health check
		storeCheck := observability.NewStoreHealthCheck("store", func(ctx context.Context) error {
			// Simple ping check - try to list something
			_, err := s.store.List(ctx, "health_check")
			if err != nil && err.Error() != "bucket not found" && err.Error() != "not found" {
				return err
			}
			return nil
		})
		s.healthChecker.RegisterCheck(storeCheck)
	}

	// Initialize Rate Limiter
	if s.config.RateLimit.Enabled {
		s.rateLimiter = ratelimit.NewLimiterFromConfig(ratelimit.Config{
			Enabled:       s.config.RateLimit.Enabled,
			RequestsPerIP: s.config.RateLimit.RequestsPerIP,
			WindowSize:    s.config.RateLimit.WindowSize,
			BurstSize:     s.config.RateLimit.BurstSize,
			Algorithm:     "token_bucket", // 默认使用令牌桶
		})
	}

	// Initialize Tracing
	if s.config.Observability.Tracing.Enabled {
		tracing, err := observability.NewTracingManager(observability.TracingConfig{
			Enabled:        s.config.Observability.Tracing.Enabled,
			ServiceName:    s.config.Observability.Tracing.ServiceName,
			ServiceVersion: s.config.Observability.Tracing.ServiceVersion,
			Environment:    s.config.Observability.Tracing.Environment,
			OTLPEndpoint:   s.config.Observability.Tracing.OTLPEndpoint,
			OTLPInsecure:   s.config.Observability.Tracing.OTLPInsecure,
			SamplingRate:   s.config.Observability.Tracing.SamplingRate,
		})
		if err != nil {
			// Log error but don't fail server startup
			fmt.Printf("Failed to initialize tracing: %v\n", err)
		} else {
			s.tracing = tracing
		}
	}
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request ID middleware
	s.router.Use(requestIDMiddleware())

	// Tracing middleware (should be early in the chain)
	if s.config.Observability.Enabled && s.config.Observability.Tracing.Enabled && s.tracing != nil {
		s.router.Use(s.tracing.Middleware())
	}

	// Logging middleware
	if s.config.Logging.Structured {
		s.router.Use(structuredLoggingMiddleware(s.config.Logging))
	} else {
		s.router.Use(gin.Logger())
	}

	// CORS middleware
	if s.config.CORS.Enabled {
		s.router.Use(corsMiddleware(s.config.CORS))
	}

	// Metrics middleware
	if s.config.Observability.Enabled && s.config.Observability.Metrics.Enabled && s.metrics != nil {
		s.router.Use(s.metrics.Middleware())
	}
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Health check endpoint (no auth required)
	if s.config.Observability.HealthCheck.Enabled {
		s.router.GET(s.config.Observability.HealthCheck.Endpoint, s.healthCheck)
	}

	// Metrics endpoint (no auth required)
	if s.config.Observability.Metrics.Enabled {
		s.router.GET(s.config.Observability.Metrics.Endpoint, s.metricsHandler)
	}

	// API v1 routes (with authentication)
	v1 := s.router.Group("/v1")

	// Apply authentication middleware
	if s.config.Auth.APIKey.Enabled {
		v1.Use(apiKeyAuthMiddleware(s.config.Auth.APIKey))
	}
	if s.config.Auth.JWT.Enabled {
		v1.Use(jwtAuthMiddleware(s.config.Auth.JWT))
	}

	// Apply rate limiting
	if s.config.RateLimit.Enabled && s.rateLimiter != nil {
		v1.Use(ratelimit.Middleware(ratelimit.Config{
			Enabled:       s.config.RateLimit.Enabled,
			RequestsPerIP: s.config.RateLimit.RequestsPerIP,
			WindowSize:    s.config.RateLimit.WindowSize,
			BurstSize:     s.config.RateLimit.BurstSize,
		}, s.rateLimiter))
	}

	// Register API routes
	s.registerAgentRoutes(v1)
	s.registerMemoryRoutes(v1)
	s.registerSessionRoutes(v1)
	s.registerWorkflowRoutes(v1)
	s.registerToolRoutes(v1)
	s.registerMiddlewareRoutes(v1)
	s.registerTelemetryRoutes(v1)
	s.registerEvalRoutes(v1)
	s.registerMCPRoutes(v1)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	fmt.Printf("🚀 AgentSDK Server starting on %s (mode: %s)\n", addr, s.config.Mode)
	fmt.Printf("📊 Health check: http://%s%s\n", addr, s.config.Observability.HealthCheck.Endpoint)
	if s.config.Observability.Metrics.Enabled {
		fmt.Printf("📈 Metrics: http://%s%s\n", addr, s.config.Observability.Metrics.Endpoint)
	}

	// Start server
	if s.config.TLS.Enabled {
		return s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	}
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	fmt.Println("🛑 Shutting down server...")

	// Shutdown tracing
	if s.tracing != nil {
		if err := s.tracing.Shutdown(ctx); err != nil {
			fmt.Printf("⚠️  Tracing shutdown error: %v\n", err)
		}
	}

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("✅ Server stopped gracefully")
	return nil
}

// Router returns the underlying Gin router for advanced customization
func (s *Server) Router() *gin.Engine {
	return s.router
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	if s.healthChecker != nil {
		info := s.healthChecker.Check(c.Request.Context())
		c.JSON(http.StatusOK, info)
	} else {
		// Fallback to simple health check
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"version":   agentsdk.Version,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// metricsHandler handles Prometheus metrics requests
func (s *Server) metricsHandler(c *gin.Context) {
	if s.metrics != nil {
		s.metrics.Handler()(c)
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics not enabled",
		})
	}
}
