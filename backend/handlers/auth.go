package handlers

import (
	"file-transfer-backend/middleware"
	"file-transfer-backend/types"
	"file-transfer-backend/utils"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	svc  types.IAuthService
	repo types.IUserRepository
}

func NewAuthHandler(svc types.IAuthService, repo types.IUserRepository) types.IAuthHandler {
	return &AuthHandler{svc: svc, repo: repo}
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
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "An account with this email already exists."})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Registration failed."})
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "No account found with this email address."})
		case "invalid password":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Incorrect password. Please try again."})
		default:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password."})
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

func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var req struct {
		Name string `json:"name" validate:"required"`
	}
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}
	user, err := h.repo.FindByID(uid)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "user not found"))
	}
	user.Name = req.Name
	if err := h.repo.Update(user); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	return c.JSON(user)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password"     validate:"required,min=6"`
	}
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}
	user, err := h.repo.FindByID(uid)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "user not found"))
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Current password is incorrect."})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "failed"))
	}
	user.Password = string(hash)
	if err := h.repo.Update(user); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	return c.JSON(fiber.Map{"message": "Password updated successfully."})
}