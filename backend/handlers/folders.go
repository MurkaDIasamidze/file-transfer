package handlers

import (
	"file-transfer-backend/middleware"
	"file-transfer-backend/models"
	"file-transfer-backend/types"
	"file-transfer-backend/utils"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type FolderHandler struct {
	repo types.IFolderRepository
}

func NewFolderHandler(repo types.IFolderRepository) types.IFolderHandler {
	return &FolderHandler{repo: repo}
}

func (h *FolderHandler) CreateFolder(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var req struct {
		Name     string `json:"name"      validate:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}
	f := &models.Folder{UserID: uid, Name: req.Name, ParentID: req.ParentID}
	if err := h.repo.Create(f); err != nil {
		slog.Error("create folder", "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "create failed"))
	}
	return c.Status(fiber.StatusCreated).JSON(f)
}

func (h *FolderHandler) ListFolders(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var parentID *uint
	if pid := c.Query("parent_id"); pid != "" {
		id, err := parseFolderUint(pid)
		if err == nil {
			parentID = &id
		}
	}
	folders, err := h.repo.ListByParent(uid, parentID)
	if err != nil {
		slog.Error("list folders", "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list failed"))
	}
	return c.JSON(folders)
}

func (h *FolderHandler) GetTrashedFolders(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	folders, err := h.repo.ListTrashed(uid)
	if err != nil {
		slog.Error("list trashed folders", "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list failed"))
	}
	return c.JSON(folders)
}

func (h *FolderHandler) TrashFolder(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseFolderUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	if err := h.repo.UpdateTrashed(id, uid, true); err != nil {
		slog.Error("trash folder", "id", id, "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	slog.Info("folder trashed", "id", id, "user", uid)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FolderHandler) RestoreFolder(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseFolderUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	if err := h.repo.UpdateTrashed(id, uid, false); err != nil {
		slog.Error("restore folder", "id", id, "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	slog.Info("folder restored", "id", id, "user", uid)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FolderHandler) DeleteFolder(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseFolderUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	if err := h.repo.Delete(id, uid); err != nil {
		slog.Error("delete folder", "id", id, "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "delete failed"))
	}
	slog.Info("folder deleted", "id", id, "user", uid)
	return c.SendStatus(fiber.StatusNoContent)
}

func parseFolderUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}