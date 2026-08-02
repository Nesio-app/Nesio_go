package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleEvents(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	channel := "nesio:events:" + userID.String()
	pubsub := s.store.RDB.Subscribe(c.Request().Context(), channel)
	defer pubsub.Close()

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set(echo.HeaderConnection, "keep-alive")
	res.WriteHeader(http.StatusOK)

	flusher, ok := res.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming unsupported")
	}

	fmt.Fprintf(res, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	flusher.Flush()

	messages := pubsub.Channel()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-heartbeat.C:
			fmt.Fprintf(res, ": ping\n\n")
			flusher.Flush()
		case msg := <-messages:
			if msg == nil {
				continue
			}
			fmt.Fprintf(res, "event: today_card\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}
