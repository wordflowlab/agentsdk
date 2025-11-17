package server

import "github.com/gin-gonic/gin"

// Option is a function that configures a Server
type Option func(*Server)

// WithCustomRouter allows setting a custom Gin router
func WithCustomRouter(setupFunc func(*Server)) Option {
	return func(s *Server) {
		setupFunc(s)
	}
}

// WithMiddleware adds custom gin middlewares to the router
func WithMiddleware(middlewares ...gin.HandlerFunc) Option {
	return func(s *Server) {
		for _, mw := range middlewares {
			s.router.Use(mw)
		}
	}
}
