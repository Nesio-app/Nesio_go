package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Nesio-app/Nesio_go/internal/models"
	"github.com/Nesio-app/Nesio_go/internal/signal"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SignalRequest struct {
	Source    string         `json:"source" validate:"required"`
	AnchorID  string         `json:"anchor_id" validate:"required"`
	Fields    map[string]any `json:"fields"`
	RawData   string         `json:"raw_data"`
	Timestamp string         `json:"timestamp,omitempty"`
}

func (s *Server) handleCreateSignal(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	var req SignalRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	signalData := models.Signal{
		Source:    req.Source,
		AnchorID:  req.AnchorID,
		Fields:    req.Fields,
		RawData:   req.RawData,
		Timestamp: time.Now().UTC(),
	}
	if req.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			signalData.Timestamp = parsed
		}
	}

	processor := signal.NewProcessor(s.store.DB)
	card, err := processor.Process(context.Background(), userID, signalData)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, card)
}
