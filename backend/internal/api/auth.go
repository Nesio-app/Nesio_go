package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/Nesio-app/Nesio_go/internal/auth"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (s *Server) handleRegister(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	id, err := auth.CreateUser(s.store.DB, req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, "user exists or invalid data")
	}
	token, err := auth.GenerateToken(id, req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"token": token,
		"user": map[string]any{"id": id, "email": req.Email},
	})
}

func (s *Server) handleLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	id, err := auth.AuthenticateUser(s.store.DB, req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}
	token, err := auth.GenerateToken(id, req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{"id": id, "email": req.Email},
	})
}
