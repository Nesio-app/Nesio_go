package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/Nesio-app/Nesio_go/internal/auth"
	"github.com/redis/go-redis/v9"
)

func JWTAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
		}
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		return next(c)
	}
}

func RateLimit(rdb *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Simple rate limit: 100 req/min per authenticated user, fallback to IP.
			identifier := c.RealIP()
			if userID, ok := c.Get("user_id").(string); ok && userID != "" {
				identifier = userID
			}
			if identifier == c.RealIP() {
				if userID, ok := c.Get("user_id").(interface{ String() string }); ok {
					identifier = userID.String()
				}
			}
			key := "ratelimit:" + identifier
			ctx := c.Request().Context()
			pipe := rdb.Pipeline()
			incr := pipe.Incr(ctx, key)
			pipe.Expire(ctx, key, 60*time.Second)
			_, _ = pipe.Exec(ctx)
			c.Response().Header().Set("X-RateLimit-Limit", "100")
			c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, 100-incr.Val())))
			if incr.Val() > 100 {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
