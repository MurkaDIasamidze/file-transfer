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
			slog.Warn("jwt auth failed", "path", c.Path(), "ip", c.IP(), "err", err.Error())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		},
	})
}

// WSJWTMiddleware validates JWT from ?token= query param for WebSocket upgrades.
// Browsers cannot send custom headers during the WS handshake, so the token
// must travel as a query param.
//
// The resolved user ID is stored as a plain uint in Locals["ws_user_id"].
// We do NOT store the *fiber.Ctx itself because fasthttp hijacks the connection
// after the upgrade; the original Ctx is recycled and becomes invalid, so any
// pointer to it will cause a nil-dereference panic at runtime.
func WSJWTMiddleware(cfg *config.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			if auth := c.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				tokenStr = auth[7:]
			}
		}
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token required"})
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.Secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid claims"})
		}
		uidFloat, ok := claims["user_id"].(float64)
		if !ok || uidFloat == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user_id in token"})
		}

		// Store the resolved uint — this value survives the fasthttp hijack
		c.Locals("ws_user_id", uint(uidFloat))
		return c.Next()
	}
}

// UserIDFromToken reads the user ID set by the standard JWTMiddleware (REST routes).
func UserIDFromToken(c *fiber.Ctx) uint {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	return uint(claims["user_id"].(float64))
}

// WSUserID reads the user ID stored by WSJWTMiddleware.
// Pass conn.Locals as the argument inside a websocket.New(...) handler.
func WSUserID(locals func(key string) interface{}) uint {
	v := locals("ws_user_id")
	if v == nil {
		return 0
	}
	id, _ := v.(uint)
	return id
}

func GenerateToken(cfg *config.JWTConfig, userID uint, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Duration(cfg.ExpiryHours) * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.Secret))
}

func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
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