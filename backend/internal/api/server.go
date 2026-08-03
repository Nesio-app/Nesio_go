package api

import (
	"context"
	"os"

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
		return c.JSON(200, map[string]string{
			"status":   "ok",
			"revision": os.Getenv("RAILWAY_GIT_COMMIT_SHA"),
		})
	})
	s.e.POST("/api/v1/auth/register", s.handleRegister)
	s.e.POST("/api/v1/auth/login", s.handleLogin)
	s.e.POST("/api/v1/auth/forgot-password", s.handleForgotPassword)
	s.e.POST("/api/v1/auth/reset-password", s.handleResetPassword)

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

	// Item system
	api.GET("/rooms", s.handleListRooms)
	api.POST("/rooms", s.handleCreateRoom)
	api.PATCH("/rooms/:id", s.handleUpdateRoom)
	api.DELETE("/rooms/:id", s.handleDeleteRoom)

	api.GET("/containers", s.handleListContainers)
	api.POST("/containers", s.handleCreateContainer)
	api.PATCH("/containers/:id", s.handleUpdateContainer)
	api.DELETE("/containers/:id", s.handleDeleteContainer)

	api.GET("/items", s.handleListItems)
	api.POST("/items/analyze", s.handleAnalyzeItem)
	api.POST("/vision/analyze", s.handleAnalyzeItem)
	api.POST("/items/create", s.handleCreateItem)
	api.GET("/items/where-is", s.handleWhereIsItem)
	api.POST("/items/where-is-photo", s.handleWhereIsItemPhoto)
	api.GET("/items/expiring", s.handleListExpiringItems)
	api.GET("/items/documents", s.handleListDocuments)
	api.POST("/items/:id/snooze-expiry", s.handleSnoozeExpiry)
	api.GET("/items/:id", s.handleGetItem)
	api.PATCH("/items/:id", s.handleUpdateItem)
	api.DELETE("/items/:id", s.handleDeleteItem)

	// Reminder, medications, daily brief
	api.GET("/reminders", s.handleListReminders)
	api.POST("/reminders", s.handleCreateReminder)
	api.POST("/reminders/:id/done", s.handleDoneReminder)
	api.GET("/medications", s.handleListMedications)
	api.POST("/medications", s.handleCreateMedication)
	api.GET("/daily-briefs/today", s.handleGetDailyBrief)
	api.POST("/daily-briefs/generate", s.handleGenerateDailyBrief)

	// Smart intake from center input
	api.POST("/intake/ingest", s.handleIntakeIngest)
	api.POST("/intake/upload", s.handleIntakeUpload)

	// Search and relations (3.0 foundation)
	api.GET("/search", s.handleSearch)
	api.GET("/nodes/mention", s.handleNodesMention)
	api.GET("/relations", s.handleListRelations)
	api.POST("/relations", s.handleCreateRelation)
	api.DELETE("/relations/:id", s.handleDeleteRelation)

	// Chat
	api.POST("/chat", s.handleChat)
	api.POST("/ask", s.handleAsk)
	api.GET("/chat/history", s.handleChatHistory)
	api.GET("/events", s.handleEvents)

	// Extraction compatibility routes
	api.POST("/extraction/analyze", s.handleExtractionAnalyze)
	api.POST("/extraction/upload", s.handleExtractionUpload)

	// Connectors
	api.GET("/connectors", s.handleListConnectors)
	api.GET("/connectors/providers", s.handleListConnectorProviders)
	api.POST("/connectors/:provider/auth", s.handleConnectorAuth)
	api.POST("/connectors/:provider/import", s.handleImportConnectorProvider)
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

	// Additional compatibility aliases
	api.GET("/medicine", s.handleListMedications)
	api.POST("/medicine/ocr", s.handleMedicineOCR)
	api.POST("/medicine/:id/reminder", s.handleMedicineReminder)
	api.GET("/daily-brief", s.handleDailyBriefAlias)
	api.POST("/daily-brief/:id/read", s.handleMarkDailyBriefRead)
	api.POST("/export", s.handleExportData)
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}
