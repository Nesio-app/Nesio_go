package api

import (
	"context"

	"github.com/Nesio-app/Nesio_go/internal/middleware"
	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

type Server struct {
	e     *echo.Echo
	store *storage.Store
}

func NewServer(store *storage.Store) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	s := &Server{e: e, store: store}
	e.Use(middleware.RateLimit(store.RDB))
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Public
	s.e.GET("/health", func(c echo.Context) error {
		if err := s.store.RDB.Ping(c.Request().Context()).Err(); err != nil {
			return c.JSON(503, map[string]string{"status": "unhealthy", "redis": err.Error()})
		}
		if err := s.store.DB.PingContext(c.Request().Context()); err != nil {
			return c.JSON(503, map[string]string{"status": "unhealthy", "db": err.Error()})
		}
		return c.JSON(200, map[string]string{"status": "ok"})
	})
	s.e.POST("/api/v1/auth/register", s.handleRegister)
	s.e.POST("/api/v1/auth/login", s.handleLogin)

	// Public OAuth callbacks
	s.e.GET("/api/v1/connectors/:provider/authorize", s.handleConnectorAuthorize)
	s.e.GET("/api/v1/connectors/:provider/callback", s.handleConnectorCallback)
	s.e.GET("/api/v1/connectors/gmail/oauth/callback", s.handleGmailOAuthCallback)

	// Protected
	api := s.e.Group("/api/v1")
	api.Use(middleware.JWTAuth)

	// Today
	api.GET("/today", s.handleGetToday)
	api.POST("/cards/:id/dismiss", s.handleDismissCard)
	api.POST("/cards/:id/mute", s.handleMuteCard)
	api.POST("/cards/:id/done", s.handleDoneCard)

	// Tasks
	api.POST("/tasks", s.handleCreateTask)
	api.PATCH("/tasks/:id", s.handleUpdateTask)
	api.GET("/tasks", s.handleListTasks)

	// Chat
	api.POST("/chat", s.handleChat)
	api.GET("/chat/history", s.handleChatHistory)
	api.GET("/events", s.handleEvents)

	// Connectors
	api.GET("/connectors", s.handleListConnectors)
	api.POST("/connectors/:provider/auth", s.handleConnectorAuth)
	api.DELETE("/connectors/:id", s.handleDeleteConnector)
	api.POST("/connectors/:id/sync", s.handleSyncConnector)
	api.GET("/connectors/gmail/oauth/authorize", s.handleGmailOAuthAuthorize)
	api.GET("/connectors/gmail/inbox", s.handleGmailInbox)
	api.POST("/connectors/gmail/send", s.handleGmailSend)

	// Signal ingestion
	api.POST("/signals", s.handleCreateSignal)

	// Memory
	api.GET("/memories", s.handleListMemories)
	api.POST("/memories", s.handleCreateMemory)
	api.GET("/domains/overview", s.handleDomainsOverview)
	api.GET("/domains/:domain/detail", s.handleDomainDetail)
	api.POST("/domains/:domain/tasks", s.handleCreateDomainTask)
	api.POST("/domains/:domain/memories", s.handleCreateDomainMemory)
	api.PATCH("/domains/:domain/nodes/:id", s.handleUpdateDomainNode)
	api.DELETE("/domains/:domain/nodes/:id", s.handleDeleteDomainNode)

	// User
	api.GET("/me", s.handleGetMe)
	api.PATCH("/me", s.handleUpdateMe)
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}
