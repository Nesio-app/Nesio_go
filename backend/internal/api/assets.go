package api

import (
	"database/sql"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const maxAssetBytes int64 = 15 << 20

type assetResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type storedAsset struct {
	Filename    string `db:"filename"`
	ContentType string `db:"content_type"`
	Data        []byte `db:"data"`
}

func (s *Server) handleUploadAsset(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}
	if file.Size <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "file must not be empty")
	}
	if file.Size > maxAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "file exceeds 15 MiB limit")
	}

	source, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "unable to read uploaded file")
	}
	defer source.Close()

	data, err := io.ReadAll(io.LimitReader(source, maxAssetBytes+1))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "unable to read uploaded file")
	}
	if int64(len(data)) > maxAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "file exceeds 15 MiB limit")
	}

	contentType := supportedAssetContentType(file.Header.Get(echo.HeaderContentType), data)
	if contentType == "" {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, "unsupported file type")
	}

	var assetID uuid.UUID
	if err := s.store.DB.QueryRow(`
		INSERT INTO assets (user_id, filename, content_type, byte_size, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, userID, file.Filename, contentType, len(data), data).Scan(&assetID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, assetResponse{
		ID:  assetID.String(),
		URL: "assets/" + assetID.String(),
	})
}

func (s *Server) handleGetAsset(c echo.Context) error {
	userID := c.Get("user_id").(uuid.UUID)
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid asset id")
	}

	var asset storedAsset
	if err := s.store.DB.Get(&asset, `
		SELECT filename, content_type, data
		FROM assets
		WHERE id = $1 AND user_id = $2
	`, assetID, userID); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "asset not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "private, max-age=300")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	if disposition := mime.FormatMediaType("inline", map[string]string{"filename": asset.Filename}); disposition != "" {
		c.Response().Header().Set(echo.HeaderContentDisposition, disposition)
	}
	return c.Blob(http.StatusOK, asset.ContentType, asset.Data)
}

func supportedAssetContentType(declared string, data []byte) string {
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(declared, ";")[0]))
	if isSupportedAssetContentType(contentType) {
		return contentType
	}
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "text/") {
		return ""
	}

	contentType = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
	if isSupportedAssetContentType(contentType) {
		return contentType
	}
	return ""
}

func isSupportedAssetContentType(contentType string) bool {
	switch contentType {
	case "application/pdf",
		"image/gif",
		"image/heic",
		"image/heif",
		"image/jpeg",
		"image/png",
		"image/webp",
		"text/markdown",
		"text/plain":
		return true
	default:
		return false
	}
}
