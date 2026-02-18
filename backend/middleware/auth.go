package middleware

import (
	"file-transfer-backend/config"
	"log/slog"
	"strings"
	"time"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(cfg *config.JWTConfig) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(cfg.Secret)},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			slog.Warn("jwt auth failed",
				"path", c.Path(),
				"ip",   c.IP(),
				"err",  err.Error(),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		},
	})
}

// UserIDFromToken extracts the numeric user id from the JWT claims
func UserIDFromToken(c *fiber.Ctx) uint {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	return uint(claims["user_id"].(float64))
}

// GenerateToken creates a signed JWT for the given user
func GenerateToken(cfg *config.JWTConfig, userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Duration(cfg.ExpiryHours) * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(cfg.Secret))
}

// Logger is a simple slog-based request logger middleware
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// skip logging websocket upgrade paths
		if strings.HasPrefix(c.Path(), "/ws") {
			return err
		}

		slog.Info("http",
			"method",  c.Method(),
			"path",    c.Path(),
			"status",  c.Response().StatusCode(),
			"latency", time.Since(start).String(),
			"ip",      c.IP(),
		)
		return err
	}
}