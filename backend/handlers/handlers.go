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
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ─── Message types ────────────────────────────────────────────────────────────

type wsMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// client → server
type initMsg struct {
	FileName  string `json:"file_name"`
	FileType  string `json:"file_type"`
	FileSize  int64  `json:"file_size"`
	Checksum  string `json:"checksum"`
	FolderID  *uint  `json:"folder_id"`
	RelPath   string `json:"rel_path"` // for folder uploads, e.g. "docs/report.pdf"
}

type chunkMsg struct {
	FileUploadID uint   `json:"file_upload_id"`
	ChunkIndex   int    `json:"chunk_index"`
	TotalChunks  int    `json:"total_chunks"`
	Checksum     string `json:"checksum"`
	Data         string `json:"data"` // base64-encoded bytes
}

type completeMsg struct {
	FileUploadID uint `json:"file_upload_id"`
}

// server → client
type progressMsg struct {
	Type         string  `json:"type"`
	FileUploadID uint    `json:"file_upload_id"`
	FileName     string  `json:"file_name"`
	Uploaded     int     `json:"uploaded_chunks"`
	Total        int     `json:"total_chunks"`
	Percent      float64 `json:"progress_percent"`
	Status       string  `json:"status"`
}

type errorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type doneMsg struct {
	Type string             `json:"type"`
	File *models.FileUpload `json:"file"`
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type UploadWSHandler struct {
	repo types.IFileRepository
	cs   types.IChecksumService
	fs   types.IFileService
	cfg  *config.UploadConfig
	mu   sync.Mutex
}

func NewUploadWSHandler(
	repo types.IFileRepository,
	cs types.IChecksumService,
	fs types.IFileService,
	cfg *config.UploadConfig,
) *UploadWSHandler {
	return &UploadWSHandler{repo: repo, cs: cs, fs: fs, cfg: cfg}
}

// HandleUpload is the WebSocket handler mounted at /ws/upload
func (h *UploadWSHandler) HandleUpload(conn *websocket.Conn) {
	// Authenticate via query-string token (WS can't set headers)
	// The JWT middleware already validated the token before upgrade,
	// so we read the user ID stored in Locals by the middleware.
	uid := middleware.WSUserID(conn.Locals)

	slog.Info("ws upload connected", "user", uid)
	defer conn.Close()

	// Per-connection state
	chunks := make(map[uint]map[int][]byte) // fileID → chunkIdx → data
	totals := make(map[uint]int)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			slog.Info("ws upload disconnected", "user", uid)
			return
		}

		var msg wsMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			h.sendError(conn, "invalid message format")
			continue
		}

		switch msg.Type {
		case "init":
			h.handleInit(conn, uid, msg.Data, chunks, totals)
		case "chunk":
			h.handleChunk(conn, uid, msg.Data, chunks, totals)
		case "complete":
			h.handleComplete(conn, uid, msg.Data, chunks)
		default:
			h.sendError(conn, "unknown message type: "+msg.Type)
		}
	}
}

func (h *UploadWSHandler) handleInit(
	conn *websocket.Conn, uid uint,
	data json.RawMessage,
	chunks map[uint]map[int][]byte,
	totals map[uint]int,
) {
	var req initMsg
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(conn, "invalid init payload")
		return
	}

	fu := &models.FileUpload{
		UserID:   uid,
		FolderID: req.FolderID,
		FileName: req.FileName,
		FileType: req.FileType,
		FileSize: req.FileSize,
		Checksum: req.Checksum,
		Status:   "pending",
		RelPath:  req.RelPath,
	}

	// For folder uploads: TotalChunks is set when we receive the first chunk
	// We'll update it on first chunk arrival. Start at 0.
	if err := h.repo.Create(fu); err != nil {
		slog.Error("ws init create", "err", err)
		h.sendError(conn, "failed to init upload")
		return
	}

	chunks[fu.ID] = make(map[int][]byte)
	slog.Info("ws upload init", "file", fu.ID, "name", req.FileName)

	conn.WriteJSON(map[string]interface{}{
		"type":           "init_ack",
		"file_upload_id": fu.ID,
		"file_name":      fu.FileName,
	})
}

func (h *UploadWSHandler) handleChunk(
	conn *websocket.Conn, uid uint,
	data json.RawMessage,
	chunks map[uint]map[int][]byte,
	totals map[uint]int,
) {
	var req chunkMsg
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(conn, "invalid chunk payload")
		return
	}

	// Decode base64 payload
	rawData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		h.sendError(conn, fmt.Sprintf("invalid base64 in chunk %d", req.ChunkIndex))
		return
	}

	// Verify checksum
	sum := sha256.Sum256(rawData)
	actual := hex.EncodeToString(sum[:])
	if actual != req.Checksum {
		h.sendError(conn, fmt.Sprintf("checksum mismatch chunk %d", req.ChunkIndex))
		return
	}

	fileChunks, ok := chunks[req.FileUploadID]
	if !ok {
		h.sendError(conn, "unknown file_upload_id — send init first")
		return
	}

	fileChunks[req.ChunkIndex] = rawData
	totals[req.FileUploadID] = req.TotalChunks

	// Update status to uploading
	if fu, err := h.repo.GetByID(req.FileUploadID); err == nil {
		if fu.TotalChunks == 0 {
			fu.TotalChunks = req.TotalChunks
		}
		fu.Status = "uploading"
		h.repo.Update(fu)
	}

	uploaded := len(fileChunks)
	pct := float64(uploaded) / float64(req.TotalChunks) * 100

	conn.WriteJSON(progressMsg{
		Type:         "progress",
		FileUploadID: req.FileUploadID,
		Uploaded:     uploaded,
		Total:        req.TotalChunks,
		Percent:      pct,
		Status:       "uploading",
	})
}

func (h *UploadWSHandler) handleComplete(
	conn *websocket.Conn, uid uint,
	data json.RawMessage,
	chunks map[uint]map[int][]byte,
) {
	var req completeMsg
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(conn, "invalid complete payload")
		return
	}

	fu, err := h.repo.GetByID(req.FileUploadID)
	if err != nil {
		h.sendError(conn, "file upload not found")
		return
	}
	if fu.UserID != uid {
		h.sendError(conn, "forbidden")
		return
	}

	fileChunks, ok := chunks[req.FileUploadID]
	if !ok || len(fileChunks) == 0 {
		h.sendError(conn, "no chunks received")
		return
	}

	total := fu.TotalChunks
	if total == 0 {
		total = len(fileChunks)
		fu.TotalChunks = total
	}

	// Build output path — respect rel_path for folder uploads
	userDir := filepath.Join(h.cfg.Directory, strconv.FormatUint(uint64(uid), 10))
	if fu.FolderID != nil {
		userDir = filepath.Join(userDir, strconv.FormatUint(uint64(*fu.FolderID), 10))
	}

	var outPath string
	if fu.RelPath != "" {
		outPath = filepath.Join(userDir, filepath.FromSlash(fu.RelPath))
	} else {
		outPath = filepath.Join(userDir, fu.FileName)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), os.ModePerm); err != nil {
		h.sendError(conn, "mkdir failed")
		return
	}

	// Reconstruct from in-memory chunks
	f, err := os.Create(outPath)
	if err != nil {
		h.sendError(conn, "create file failed")
		return
	}

	for i := 0; i < total; i++ {
		chunk, found := fileChunks[i]
		if !found {
			f.Close()
			h.sendError(conn, fmt.Sprintf("missing chunk %d", i))
			return
		}
		f.Write(chunk)
	}
	f.Close()

	// Verify whole-file checksum
	fileData, err := os.ReadFile(outPath)
	if err != nil {
		h.sendError(conn, "read file failed")
		return
	}
	sum := sha256.Sum256(fileData)
	actual := hex.EncodeToString(sum[:])
	if actual != fu.Checksum {
		fu.Status = "failed"
		h.repo.Update(fu)
		h.sendError(conn, "file checksum mismatch")
		return
	}

	fu.Status = "completed"
	fu.FilePath = outPath
	h.repo.Update(fu)

	// Free memory
	delete(chunks, req.FileUploadID)

	slog.Info("ws upload complete", "file", fu.ID, "name", fu.FileName)

	conn.WriteJSON(doneMsg{Type: "done", File: fu})
}

func (h *UploadWSHandler) sendError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(errorMsg{Type: "error", Message: msg})
}

// ─── REST fallback (for verify) ───────────────────────────────────────────────

type FileHandler struct {
	repo      types.IFileRepository
	cs        types.IChecksumService
	fs        types.IFileService
	cfg       *config.UploadConfig
}

func NewFileHandler(
	repo types.IFileRepository,
	cs types.IChecksumService,
	fs types.IFileService,
	cfg *config.UploadConfig,
) types.IFileHandler {
	return &FileHandler{repo: repo, cs: cs, fs: fs, cfg: cfg}
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
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	var req struct {
		FolderID *uint `json:"folder_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid body"))
	}
	file, err := h.repo.GetByID(id)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "file not found"))
	}
	if file.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}
	if err := h.repo.UpdateFolderID(id, req.FolderID); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	file.FolderID = req.FolderID
	return c.JSON(file)
}

func (h *FileHandler) ToggleStar(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	file, err := h.repo.GetByID(id)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "file not found"))
	}
	if file.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}
	file.Starred = !file.Starred
	if err := h.repo.Update(file); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	return c.JSON(file)
}

func (h *FileHandler) TrashFile(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	file, err := h.repo.GetByID(id)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "file not found"))
	}
	if file.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}
	if err := h.repo.UpdateTrashed(id, true); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FileHandler) RestoreFile(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	id, err := parseUint(c.Params("id"))
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id"))
	}
	file, err := h.repo.GetByID(id)
	if err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusNotFound, "file not found"))
	}
	if file.UserID != uid {
		return utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden"))
	}
	if err := h.repo.UpdateTrashed(id, false); err != nil {
		return utils.Respond(c, utils.NewError(fiber.StatusInternalServerError, "update failed"))
	}
	file.Trashed = false
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

// Stub methods to satisfy IFileHandler (WS-based upload replaces these)
func (h *FileHandler) InitUpload(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "use WebSocket upload"})
}
func (h *FileHandler) UploadChunk(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "use WebSocket upload"})
}
func (h *FileHandler) CompleteUpload(c *fiber.Ctx) error {
	return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "use WebSocket upload"})
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
func (h *FileHandler) HandleWebSocket(conn *websocket.Conn) { conn.Close() }

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}

// healthCheck used in main
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}