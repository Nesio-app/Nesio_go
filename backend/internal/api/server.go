package api

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/Nesio-app/Nesio_go/internal/storage"
)

type Server struct {
	e     *echo.Echo
	store *storage.Store
}

func NewServer(store *storage.Store) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	s := &Server{e: e, store: store}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.e.POST("/api/v1/auth/register", s.handleRegister)
	s.e.POST("/api/v1/auth/login", s.handleLogin)
	api := s.e.Group("/api/v1")
	api.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte("nesio_dev_secret_change_in_prod"),
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return path == "/api/v1/auth/register" || path == "/api/v1/auth/login"
		},
	}))
	api.GET("/today", s.handleGetToday)
	api.POST("/cards/:id/dismiss", s.handleDismissCard)
	api.POST("/cards/:id/mute", s.handleMuteCard)
	api.POST("/cards/:id/done", s.handleDoneCard)
	api.POST("/tasks", s.handleCreateTask)
	api.PATCH("/tasks/:id", s.handleUpdateTask)
	api.GET("/tasks", s.handleListTasks)
	api.POST("/chat", s.handleChat)
	api.GET("/chat/history", s.handleChatHistory)
	api.GET("/connectors", s.handleListConnectors)
	api.POST("/connectors/:provider/auth", s.handleConnectorAuth)
	api.DELETE("/connectors/:id", s.handleDeleteConnector)
	api.POST("/connectors/:id/sync", s.handleSyncConnector)
	api.GET("/me", s.handleGetMe)
	api.PATCH("/me", s.handleUpdateMe)
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}
