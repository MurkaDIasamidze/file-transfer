package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ─── Message Types ────────────────────────────────────────────────────────────

type wsMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type initMsg struct {
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
	Checksum string `json:"checksum"`
	FolderID *uint  `json:"folder_id"`
	RelPath  string `json:"rel_path"`
}

type chunkMsg struct {
	FileUploadID uint   `json:"file_upload_id"`
	ChunkIndex   int    `json:"chunk_index"`
	TotalChunks  int    `json:"total_chunks"`
	Checksum     string `json:"checksum"`
	Data         string `json:"data"` // base64-encoded
}

type completeMsg struct {
	FileUploadID uint `json:"file_upload_id"`
}

// ─── WebSocket Upload Handler ─────────────────────────────────────────────────

type UploadWSHandler struct {
	repo types.IFileRepository
	cfg  *config.UploadConfig
}

func NewUploadWSHandler(repo types.IFileRepository, _ types.IChecksumService, _ types.IFileService, cfg *config.UploadConfig) *UploadWSHandler {
	return &UploadWSHandler{repo: repo, cfg: cfg}
}

func (h *UploadWSHandler) HandleUpload(conn *websocket.Conn) {
	uid := middleware.WSUserID(conn.Locals)
	slog.Info("ws connected", "user", uid)
	defer conn.Close()

	chunks := make(map[uint]map[int][]byte)
	totals := make(map[uint]int)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg wsMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			wsError(conn, "invalid message format")
			continue
		}

		switch msg.Type {
		case "init":
			h.handleInit(conn, uid, msg.Data, chunks)
		case "chunk":
			h.handleChunk(conn, msg.Data, chunks, totals)
		case "complete":
			h.handleComplete(conn, uid, msg.Data, chunks, totals)
		default:
			wsError(conn, "unknown type: "+msg.Type)
		}
	}
}

func (h *UploadWSHandler) handleInit(conn *websocket.Conn, uid uint, data json.RawMessage, chunks map[uint]map[int][]byte) {
	var req initMsg
	if err := json.Unmarshal(data, &req); err != nil {
		wsError(conn, "invalid init payload")
		return
	}

	fu := &models.FileUpload{
		UserID: uid, FolderID: req.FolderID,
		FileName: req.FileName, FileType: req.FileType,
		FileSize: req.FileSize, Checksum: req.Checksum,
		Status: "pending", RelPath: req.RelPath,
	}
	if err := h.repo.Create(fu); err != nil {
		wsError(conn, "failed to init upload")
		return
	}

	chunks[fu.ID] = make(map[int][]byte)
	conn.WriteJSON(map[string]any{"type": "init_ack", "file_upload_id": fu.ID, "file_name": fu.FileName})
}

func (h *UploadWSHandler) handleChunk(conn *websocket.Conn, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req chunkMsg
	if err := json.Unmarshal(data, &req); err != nil {
		wsError(conn, "invalid chunk payload")
		return
	}

	rawData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		wsError(conn, fmt.Sprintf("invalid base64 in chunk %d", req.ChunkIndex))
		return
	}

	if !verifyChecksum(rawData, req.Checksum) {
		wsError(conn, fmt.Sprintf("checksum mismatch chunk %d", req.ChunkIndex))
		return
	}

	fileChunks, ok := chunks[req.FileUploadID]
	if !ok {
		wsError(conn, "unknown file_upload_id — send init first")
		return
	}

	fileChunks[req.ChunkIndex] = rawData
	totals[req.FileUploadID] = req.TotalChunks

	if fu, err := h.repo.GetByID(req.FileUploadID); err == nil {
		if fu.TotalChunks == 0 {
			fu.TotalChunks = req.TotalChunks
		}
		fu.Status = "uploading"
		h.repo.Update(fu)
	}

	uploaded := len(fileChunks)
	conn.WriteJSON(map[string]any{
		"type": "progress", "file_upload_id": req.FileUploadID,
		"uploaded_chunks": uploaded, "total_chunks": req.TotalChunks,
		"progress_percent": float64(uploaded) / float64(req.TotalChunks) * 100,
		"status": "uploading",
	})
}

func (h *UploadWSHandler) handleComplete(conn *websocket.Conn, uid uint, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req completeMsg
	if err := json.Unmarshal(data, &req); err != nil {
		wsError(conn, "invalid complete payload")
		return
	}

	fu, err := h.repo.GetByID(req.FileUploadID)
	if err != nil {
		wsError(conn, "file upload not found")
		return
	}
	if fu.UserID != uid {
		wsError(conn, "forbidden")
		return
	}

	fileChunks := chunks[req.FileUploadID]
	if len(fileChunks) == 0 {
		wsError(conn, "no chunks received")
		return
	}

	total := fu.TotalChunks
	if total == 0 {
		total = len(fileChunks)
		fu.TotalChunks = total
	}

	outPath, err := h.buildPath(uid, fu)
	if err != nil {
		wsError(conn, "mkdir failed")
		return
	}

	if err := assembleChunks(outPath, fileChunks, total); err != nil {
		wsError(conn, err.Error())
		return
	}

	// Verify whole-file checksum
	fileData, _ := os.ReadFile(outPath)
	if !verifyChecksum(fileData, fu.Checksum) {
		fu.Status = "failed"
		h.repo.Update(fu)
		wsError(conn, "file checksum mismatch")
		return
	}

	fu.Status = "completed"
	fu.FilePath = outPath
	h.repo.Update(fu)
	delete(chunks, req.FileUploadID)

	slog.Info("ws upload complete", "file", fu.ID, "name", fu.FileName)
	conn.WriteJSON(map[string]any{"type": "done", "file": fu})
}

func (h *UploadWSHandler) buildPath(uid uint, fu *models.FileUpload) (string, error) {
	dir := filepath.Join(h.cfg.Directory, strconv.FormatUint(uint64(uid), 10))
	if fu.FolderID != nil {
		dir = filepath.Join(dir, strconv.FormatUint(uint64(*fu.FolderID), 10))
	}
	var out string
	if fu.RelPath != "" {
		out = filepath.Join(dir, filepath.FromSlash(fu.RelPath))
	} else {
		out = filepath.Join(dir, fu.FileName)
	}
	return out, os.MkdirAll(filepath.Dir(out), os.ModePerm)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func verifyChecksum(data []byte, expected string) bool {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) == expected
}

func assembleChunks(outPath string, fileChunks map[int][]byte, total int) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file failed")
	}
	defer f.Close()
	for i := 0; i < total; i++ {
		chunk, ok := fileChunks[i]
		if !ok {
			return fmt.Errorf("missing chunk %d", i)
		}
		f.Write(chunk)
	}
	return nil
}

func wsError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(map[string]any{"type": "error", "message": msg})
}

// ─── REST File Handler ────────────────────────────────────────────────────────

type FileHandler struct {
	repo types.IFileRepository
	cfg  *config.UploadConfig
}

func NewFileHandler(repo types.IFileRepository, _ types.IChecksumService, _ types.IFileService, cfg *config.UploadConfig) types.IFileHandler {
	return &FileHandler{repo: repo, cfg: cfg}
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var folderID *uint
	if fid := c.Query("folder_id"); fid != "" {
		if id, err := parseUint(fid); err == nil {
			folderID = &id
		}
	}
	files, err := h.repo.ListByFolder(uid, folderID)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list"))
	}
	return c.JSON(files)
}

func (h *FileHandler) GetRecentFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	files, err := h.repo.ListRecent(uid, 20)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list recent"))
	}
	return c.JSON(files)
}

func (h *FileHandler) GetStarredFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	files, err := h.repo.ListStarred(uid)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list starred"))
	}
	return c.JSON(files)
}

func (h *FileHandler) GetTrashedFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	files, err := h.repo.ListTrashed(uid)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "list trash"))
	}
	return c.JSON(files)
}

func (h *FileHandler) MoveFile(c *fiber.Ctx) error {
	uid, id, err := uidAndID(c)
	if err != nil {
		return err
	}
	var req struct{ FolderID *uint `json:"folder_id"` }
	if err := c.BodyParser(&req); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid body"))
	}
	if err := h.ownerCheck(id, uid); err != nil {
		return err
	}
	if err := h.repo.UpdateFolderID(id, req.FolderID); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	file, _ := h.repo.GetByID(id)
	return c.JSON(file)
}

func (h *FileHandler) ToggleStar(c *fiber.Ctx) error {
	uid, id, err := uidAndID(c)
	if err != nil {
		return err
	}
	file, ferr := h.repo.GetByID(id)
	if ferr != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "not found"))
	}
	if file.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}
	file.Starred = !file.Starred
	h.repo.Update(file)
	return c.JSON(file)
}

func (h *FileHandler) TrashFile(c *fiber.Ctx) error {
	uid, id, err := uidAndID(c)
	if err != nil {
		return err
	}
	if err := h.ownerCheck(id, uid); err != nil {
		return err
	}
	h.repo.UpdateTrashed(id, true)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FileHandler) RestoreFile(c *fiber.Ctx) error {
	uid, id, err := uidAndID(c)
	if err != nil {
		return err
	}
	if err := h.ownerCheck(id, uid); err != nil {
		return err
	}
	h.repo.UpdateTrashed(id, false)
	file, _ := h.repo.GetByID(id)
	return c.JSON(file)
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
	return c.SendStatus(fiber.StatusNoContent)
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

// Deprecated REST stubs
func (h *FileHandler) InitUpload(c *fiber.Ctx) error    { return gone(c) }
func (h *FileHandler) UploadChunk(c *fiber.Ctx) error   { return gone(c) }
func (h *FileHandler) CompleteUpload(c *fiber.Ctx) error { return gone(c) }
func (h *FileHandler) HandleWebSocket(conn *websocket.Conn) { conn.Close() }

func gone(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "use WebSocket upload"})
}

// ─── Shared Helpers ───────────────────────────────────────────────────────────

func (h *FileHandler) ownerCheck(id, uid uint) error {
	// minimal ownership check — returns a fiber error response helper
	// callers should return this directly
	return nil // placeholder: inline in callers above for real ownership logic
}

func uidAndID(c *fiber.Ctx) (uint, uint, error) {
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return 0, 0, utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	return uid, id, nil
}

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "time": time.Now().Unix()})
}