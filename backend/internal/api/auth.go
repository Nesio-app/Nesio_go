package api

import (
	"net/http"

	"github.com/Nesio-app/Nesio_go/internal/auth"
	"github.com/labstack/echo/v4"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordResponse struct {
	ResetToken string `json:"reset_token"`
	ExpiresAt  string `json:"expires_at"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

func (s *Server) handleRegister(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = auth.NormalizeEmail(req.Email)
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
		"user":  map[string]any{"id": id, "email": req.Email},
	})
}

func (s *Server) handleLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = auth.NormalizeEmail(req.Email)
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
		"user":  map[string]any{"id": id, "email": req.Email},
	})
}

func (s *Server) handleForgotPassword(c echo.Context) error {
	var req ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = auth.NormalizeEmail(req.Email)
	token, expiresAt, err := auth.CreatePasswordResetToken(s.store.DB, req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, ForgotPasswordResponse{
		ResetToken: token,
		ExpiresAt:  expiresAt.Format(http.TimeFormat),
	})
}

func (s *Server) handleResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Email = auth.NormalizeEmail(req.Email)
	if err := auth.ResetPasswordWithToken(s.store.DB, req.Email, req.Token, req.Password); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid reset token")
	}
	return c.NoContent(http.StatusNoContent)
}
