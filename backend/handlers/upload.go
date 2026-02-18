package handlers

import (
	"file-transfer-backend/config"
	"file-transfer-backend/middleware"
	"file-transfer-backend/models"
	"file-transfer-backend/types"
	"file-transfer-backend/utils"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type FileHandler struct {
	repo      types.IFileRepository
	cs        types.IChecksumService
	fs        types.IFileService
	cfg       *config.UploadConfig
	wsClients map[uint]map[*websocket.Conn]bool
	wsMu      sync.RWMutex
}

func NewFileHandler(
	repo types.IFileRepository,
	cs types.IChecksumService,
	fs types.IFileService,
	cfg *config.UploadConfig,
) types.IFileHandler {
	return &FileHandler{
		repo:      repo,
		cs:        cs,
		fs:        fs,
		cfg:       cfg,
		wsClients: make(map[uint]map[*websocket.Conn]bool),
	}
}

type initUploadReq struct {
	FileName    string `json:"file_name"    validate:"required"`
	FileType    string `json:"file_type"`
	FileSize    int64  `json:"file_size"    validate:"required"`
	TotalChunks int    `json:"total_chunks" validate:"required"`
	Checksum    string `json:"checksum"     validate:"required"`
	FolderID    *uint  `json:"folder_id"`
}

type completeReq struct {
	FileUploadID uint `json:"file_upload_id" validate:"required"`
}

func (h *FileHandler) InitUpload(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)

	var req initUploadReq
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}

	fu := &models.FileUpload{
		UserID:      uid,
		FolderID:    req.FolderID,
		FileName:    req.FileName,
		FileType:    req.FileType,
		FileSize:    req.FileSize,
		TotalChunks: req.TotalChunks,
		Checksum:    req.Checksum,
		Status:      "pending",
	}
	if err := h.repo.Create(fu); err != nil {
		slog.Error("init upload db", "err", err)
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "failed to init upload"))
	}

	slog.Info("upload init", "file_id", fu.ID, "user", uid, "name", fu.FileName)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"file_upload_id": fu.ID})
}

func (h *FileHandler) UploadChunk(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)

	fileID, err := parseUint(c.FormValue("file_upload_id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid file_upload_id"))
	}

	idx, err := strconv.Atoi(c.FormValue("chunk_index"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid chunk_index"))
	}

	checksum := c.FormValue("checksum")
	if checksum == "" {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "checksum required"))
	}

	fh, err := c.FormFile("chunk")
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "chunk file required"))
	}

	f, err := fh.Open()
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "open chunk"))
	}
	defer f.Close()

	data := make([]byte, fh.Size)
	if _, err = f.Read(data); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "read chunk"))
	}

	if !h.cs.Verify(data, checksum) {
		slog.Warn("chunk checksum mismatch", "file", fileID, "idx", idx, "user", uid)
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "checksum mismatch"))
	}

	existing, err := h.repo.GetChunk(fileID, idx)
	if err == nil {
		existing.Data = data
		existing.Checksum = checksum
		existing.ChunkSize = len(data)
		existing.Status = "verified"
		h.repo.UpdateChunk(existing)
	} else {
		ch := &models.FileChunk{
			FileUploadID: fileID,
			ChunkIndex:   idx,
			ChunkSize:    len(data),
			Checksum:     checksum,
			Status:       "verified",
			Data:         data,
		}
		if err := h.repo.CreateChunk(ch); err != nil {
			return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "save chunk"))
		}
	}

	if fu, e := h.repo.GetByID(fileID); e == nil {
		fu.Status = "uploading"
		h.repo.Update(fu)
	}

	go h.broadcastProgress(fileID)

	slog.Debug("chunk ok", "file", fileID, "idx", idx)
	return c.JSON(fiber.Map{"chunk_index": idx})
}

func (h *FileHandler) CompleteUpload(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)

	var req completeReq
	if err := utils.BindAndValidate(c, &req); err != nil {
		return utils.Respond(c, err)
	}

	fu, err := h.repo.GetByID(req.FileUploadID)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "upload not found"))
	}
	if fu.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}

	chunks, err := h.repo.GetChunksByFileID(req.FileUploadID)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "fetch chunks"))
	}

	if len(chunks) != fu.TotalChunks {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("need %d chunks, got %d", fu.TotalChunks, len(chunks))))
	}

	userDir := filepath.Join(h.cfg.Directory, fmt.Sprintf("%d", uid))
	if fu.FolderID != nil {
		userDir = filepath.Join(userDir, fmt.Sprintf("%d", *fu.FolderID))
	}
	_ = mkDir(userDir)

	outPath := filepath.Join(userDir, fu.FileName)

	if err := h.fs.Reconstruct(fu, chunks, outPath); err != nil {
		return utils.Respond(c, err)
	}

	ok, err := h.fs.VerifyFile(outPath, fu.Checksum)
	if err != nil || !ok {
		fu.Status = "failed"
		h.repo.Update(fu)
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "final checksum failed"))
	}

	fu.Status = "completed"
	fu.FilePath = outPath
	h.repo.Update(fu)
	go h.broadcastProgress(req.FileUploadID)

	slog.Info("upload complete", "file", fu.ID, "path", outPath)
	return c.JSON(fiber.Map{"file": fu})
}

func (h *FileHandler) VerifyChunks(c *fiber.Ctx) error {
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	idx, err := h.repo.GetVerifiedChunkIndices(id)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "verify"))
	}
	return c.JSON(fiber.Map{"uploaded_chunks": idx, "total": len(idx)})
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)

	var folderID *uint
	if fid := c.Query("folder_id"); fid != "" {
		id, err := parseUint(fid)
		if err == nil {
			folderID = &id
		}
	}

	files, err := h.repo.ListByFolder(uid, folderID)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list files"))
	}
	return c.JSON(files)
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	if err := h.repo.Delete(id, uid); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "delete"))
	}
	slog.Info("file deleted", "id", id, "user", uid)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FileHandler) HandleWebSocket(conn *websocket.Conn) {
	id, err := parseUint(conn.Params("id"))
	if err != nil {
		conn.Close()
		return
	}

	h.wsMu.Lock()
	if h.wsClients[id] == nil {
		h.wsClients[id] = make(map[*websocket.Conn]bool)
	}
	h.wsClients[id][conn] = true
	h.wsMu.Unlock()

	slog.Info("ws client connected", "file", id)

	defer func() {
		h.wsMu.Lock()
		delete(h.wsClients[id], conn)
		h.wsMu.Unlock()
		conn.Close()
		slog.Info("ws client disconnected", "file", id)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (h *FileHandler) broadcastProgress(fileID uint) {
	h.wsMu.RLock()
	clients := h.wsClients[fileID]
	h.wsMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	idx, _ := h.repo.GetVerifiedChunkIndices(fileID)
	fu, _ := h.repo.GetByID(fileID)

	pct := float64(0)
	if fu != nil && fu.TotalChunks > 0 {
		pct = float64(len(idx)) / float64(fu.TotalChunks) * 100
	}

	msg := fiber.Map{
		"type":             "progress",
		"file_upload_id":   fileID,
		"uploaded_chunks":  len(idx),
		"total_chunks":     fu.TotalChunks,
		"progress_percent": pct,
		"status":           fu.Status,
		"file_name":        fu.FileName,
	}

	for cl := range clients {
		if err := cl.WriteJSON(msg); err != nil {
			cl.Close()
			h.wsMu.Lock()
			delete(h.wsClients[fileID], cl)
			h.wsMu.Unlock()
		}
	}
}

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}

func mkDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}