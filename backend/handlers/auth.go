package handlers

import (
	"file-transfer-backend/middleware"
	"file-transfer-backend/types"
	"file-transfer-backend/utils"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc types.IAuthService
}

func NewAuthHandler(svc types.IAuthService) types.IAuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"     validate:"required"`
		Email    string `json:"email"    validate:"required,email"`
		Password string `json:"password" validate:"required,min=6"`
	}
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}

	user, err := h.svc.Register(req.Name, req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "email already in use":
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "An account with this email already exists.",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Registration failed. Please try again.",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"    validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}

	token, user, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		switch err.Error() {
		case "user not found":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No account found with this email address.",
			})
		case "invalid password":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Incorrect password. Please try again.",
			})
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password.",
			})
		}
	}

	return c.JSON(fiber.Map{"token": token, "user": user})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	user, err := h.svc.Me(uid)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "user not found"))
	}
	return c.JSON(user)
}