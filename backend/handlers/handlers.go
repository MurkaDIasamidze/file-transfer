package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"file-transfer-backend/config"
	"file-transfer-backend/middleware"
	"file-transfer-backend/models"
	"file-transfer-backend/services"
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

// ─── WebSocket message types ──────────────────────────────────────────────────

type wsMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type initMsg struct {
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	Checksum string `json:"checksum"`
	RelPath  string `json:"rel_path"`
	FileSize int64  `json:"file_size"`
	FolderID *uint  `json:"folder_id"`
}

type chunkMsg struct {
	FileUploadID uint   `json:"file_upload_id"`
	ChunkIndex   int    `json:"chunk_index"`
	TotalChunks  int    `json:"total_chunks"`
	Checksum     string `json:"checksum"`
	Data         string `json:"data"`
}

type completeMsg struct {
	FileUploadID uint `json:"file_upload_id"`
}

type progressMsg struct {
	Type         string  `json:"type"`
	FileUploadID uint    `json:"file_upload_id"`
	Uploaded     int     `json:"uploaded_chunks"`
	Total        int     `json:"total_chunks"`
	Percent      float64 `json:"progress_percent"`
	Status       string  `json:"status"`
}

// ─── WebSocket upload handler ─────────────────────────────────────────────────

type UploadWSHandler struct {
	repo types.IFileRepository
	cs   types.IChecksumService
	cfg  *config.UploadConfig
	s3   *services.S3Service // nil = local storage
}

func NewUploadWSHandler(
	repo types.IFileRepository,
	cs   types.IChecksumService,
	cfg  *config.UploadConfig,
	s3   *services.S3Service,
) *UploadWSHandler {
	return &UploadWSHandler{repo: repo, cs: cs, cfg: cfg, s3: s3}
}

func (h *UploadWSHandler) HandleUpload(conn *websocket.Conn) {
	uid := middleware.WSUserID(conn.Locals)
	defer conn.Close()

	chunks := map[uint]map[int][]byte{}
	totals := map[uint]int{}

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg wsMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			wsError(conn, "invalid message"); continue
		}
		switch msg.Type {
		case "init":     h.wsInit(conn, uid, msg.Data, chunks)
		case "chunk":    h.wsChunk(conn, msg.Data, chunks, totals)
		case "complete": h.wsComplete(conn, uid, msg.Data, chunks, totals)
		default:         wsError(conn, "unknown type: "+msg.Type)
		}
	}
}

func (h *UploadWSHandler) wsInit(conn *websocket.Conn, uid uint, data json.RawMessage, chunks map[uint]map[int][]byte) {
	var req initMsg
	if err := json.Unmarshal(data, &req); err != nil {
		slog.Error("wsInit: bad JSON", "err", err)
		wsError(conn, "bad init"); return
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
	if err := h.repo.Create(fu); err != nil {
		slog.Error("wsInit: db create failed", "err", err, "user", uid)
		wsError(conn, "init failed"); return
	}

	// ✅ LOG: upload started
	slog.Info("⬆  upload started",
		"file_id",   fu.ID,
		"file_name", fu.FileName,
		"file_size", fu.FileSize,
		"user",      uid,
	)

	chunks[fu.ID] = map[int][]byte{}
	conn.WriteJSON(map[string]any{"type": "init_ack", "file_upload_id": fu.ID, "file_name": fu.FileName})
}

func (h *UploadWSHandler) wsChunk(conn *websocket.Conn, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req chunkMsg
	if err := json.Unmarshal(data, &req); err != nil {
		slog.Error("wsChunk: bad JSON", "err", err)
		wsError(conn, "bad chunk"); return
	}

	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		slog.Error("wsChunk: base64 decode failed", "file_id", req.FileUploadID, "chunk", req.ChunkIndex, "err", err)
		wsError(conn, "bad base64"); return
	}

	if sum := sha256hex(raw); sum != req.Checksum {
		slog.Warn("wsChunk: checksum mismatch", "file_id", req.FileUploadID, "chunk", req.ChunkIndex)
		wsError(conn, fmt.Sprintf("checksum mismatch chunk %d", req.ChunkIndex)); return
	}

	fc, ok := chunks[req.FileUploadID]
	if !ok {
		slog.Error("wsChunk: unknown file_upload_id", "file_id", req.FileUploadID)
		wsError(conn, "unknown file_upload_id"); return
	}

	fc[req.ChunkIndex] = raw
	totals[req.FileUploadID] = req.TotalChunks

	if len(fc) == 1 {
		if fu, err := h.repo.GetByID(req.FileUploadID); err == nil {
			fu.Status      = "uploading"
			fu.TotalChunks = req.TotalChunks
			h.repo.Update(fu)
		}
	}

	conn.WriteJSON(progressMsg{
		Type:         "progress",
		FileUploadID: req.FileUploadID,
		Uploaded:     len(fc),
		Total:        req.TotalChunks,
		Percent:      float64(len(fc)) / float64(req.TotalChunks) * 100,
		Status:       "uploading",
	})
}

func (h *UploadWSHandler) wsComplete(conn *websocket.Conn, uid uint, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req completeMsg
	if err := json.Unmarshal(data, &req); err != nil {
		slog.Error("wsComplete: bad JSON", "err", err)
		wsError(conn, "bad complete"); return
	}

	fu, err := h.repo.GetByID(req.FileUploadID)
	if err != nil || fu.UserID != uid {
		slog.Error("wsComplete: not found or forbidden", "file_id", req.FileUploadID, "user", uid)
		wsError(conn, "not found or forbidden"); return
	}

	fc, ok := chunks[req.FileUploadID]
	if !ok || len(fc) == 0 {
		slog.Error("wsComplete: no chunks in memory", "file_id", req.FileUploadID)
		wsError(conn, "no chunks"); return
	}

	total := totals[req.FileUploadID]
	if total == 0 { total = len(fc) }

	// Assemble chunks into a single byte slice
	var buf bytes.Buffer
	for i := range total {
		chunk, ok := fc[i]
		if !ok {
			slog.Error("wsComplete: missing chunk", "file_id", req.FileUploadID, "chunk", i)
			wsError(conn, fmt.Sprintf("missing chunk %d", i)); return
		}
		buf.Write(chunk)
	}
	fileBytes := buf.Bytes()

	// Verify whole-file checksum
	if sha256hex(fileBytes) != fu.Checksum {
		fu.Status = "failed"
		h.repo.Update(fu)
		slog.Error("wsComplete: file checksum mismatch", "file_id", fu.ID, "file_name", fu.FileName)
		wsError(conn, "file checksum mismatch"); return
	}

	// Free chunk memory immediately — no longer needed
	delete(chunks, req.FileUploadID)

	if h.s3 != nil {
		// ── ASYNC S3 upload ───────────────────────────────────────────────────
		//
		// We mark the file "processing" and reply done to the client RIGHT NOW.
		// The actual S3 PutObject runs in a goroutine. The user's browser
		// immediately shows the file as "processing" in the UI; once the goroutine
		// finishes, the DB row is updated to "completed" and the next file-list
		// poll will show it normally.
		//
		// This means large files (100+ MB) don't block the WebSocket connection.

		fu.Status = "processing"
		h.repo.Update(fu)

		// Reply to client immediately — they don't wait for S3
		conn.WriteJSON(map[string]any{"type": "done", "file": fu})

		slog.Info("⏳ queued for S3 upload",
			"file_id",   fu.ID,
			"file_name", fu.FileName,
			"file_size", len(fileBytes),
		)

		// Capture everything needed by the goroutine — do NOT pass conn or fc
		fuID      := fu.ID
		fuName    := fu.FileName
		fuType    := fu.FileType
		fuRelPath := fu.RelPath
		fuFolderID := fu.FolderID
		repo      := h.repo
		s3svc     := h.s3

		go func() {
			start := time.Now()

			var folderID uint
			if fuFolderID != nil { folderID = *fuFolderID }

			relPath := fuRelPath
			if relPath == "" { relPath = fuName }

			s3Key := services.BuildKey(uid, folderID, relPath)

			if _, err := s3svc.Upload(context.Background(), s3Key, fileBytes, fuType); err != nil {
				slog.Error("✗  S3 upload failed",
					"file_id",   fuID,
					"file_name", fuName,
					"key",       s3Key,
					"err",       err,
					"elapsed",   time.Since(start).Round(time.Millisecond),
				)
				// Mark as failed in DB so the UI can show error state
				if record, dbErr := repo.GetByID(fuID); dbErr == nil {
					record.Status = "failed"
					repo.Update(record)
				}
				return
			}

			// Update DB: completed + store S3 key as FilePath
			record, dbErr := repo.GetByID(fuID)
			if dbErr != nil {
				slog.Error("✗  S3 post-upload: db fetch failed", "file_id", fuID, "err", dbErr)
				return
			}
			record.Status   = "completed"
			record.FilePath = s3Key
			repo.Update(record)

			// ✅ LOG: upload finished
			slog.Info("✓  upload complete (S3)",
				"file_id",   fuID,
				"file_name", fuName,
				"key",       s3Key,
				"elapsed",   time.Since(start).Round(time.Millisecond),
			)
		}()

	} else {
		// ── Synchronous local write ───────────────────────────────────────────
		dir := filepath.Join(h.cfg.Directory, fmt.Sprint(uid))
		if fu.FolderID != nil { dir = filepath.Join(dir, fmt.Sprint(*fu.FolderID)) }
		outPath := filepath.Join(dir, filepath.FromSlash(fu.RelPath))
		if fu.RelPath == "" { outPath = filepath.Join(dir, fu.FileName) }

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			slog.Error("wsComplete: mkdir failed", "path", filepath.Dir(outPath), "err", err)
			wsError(conn, "mkdir failed"); return
		}
		if err := os.WriteFile(outPath, fileBytes, 0o644); err != nil {
			slog.Error("wsComplete: write failed", "path", outPath, "err", err)
			wsError(conn, "write failed"); return
		}

		fu.Status   = "completed"
		fu.FilePath = outPath
		h.repo.Update(fu)

		// ✅ LOG: upload finished
		slog.Info("✓  upload complete (local)",
			"file_id",   fu.ID,
			"file_name", fu.FileName,
			"path",      outPath,
			"size",      len(fileBytes),
		)

		conn.WriteJSON(map[string]any{"type": "done", "file": fu})
	}
}

func wsError(conn *websocket.Conn, msg string) {
	slog.Warn("ws error sent to client", "message", msg)
	conn.WriteJSON(map[string]any{"type": "error", "message": msg})
}

func sha256hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// ─── REST file handler ────────────────────────────────────────────────────────

type FileHandler struct {
	repo types.IFileRepository
	cfg  *config.UploadConfig
	s3   *services.S3Service
}

func NewFileHandler(repo types.IFileRepository, cfg *config.UploadConfig, s3 *services.S3Service) *FileHandler {
	return &FileHandler{repo: repo, cfg: cfg, s3: s3}
}

func (h *FileHandler) fileOwner(c *fiber.Ctx) (*models.FileUpload, error) {
	id, err := parseUint(c.Params("id"))
	if err != nil { return nil, utils.Respond(c, utils.NewError(fiber.StatusBadRequest, "invalid id")) }
	uid := middleware.UserIDFromToken(c)
	file, err := h.repo.GetByID(id)
	if err != nil { return nil, utils.Respond(c, utils.NewError(fiber.StatusNotFound, "not found")) }
	if file.UserID != uid { return nil, utils.Respond(c, utils.NewError(fiber.StatusForbidden, "forbidden")) }
	return file, nil
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	uid := middleware.UserIDFromToken(c)
	var folderID *uint
	if fid := c.Query("folder_id"); fid != "" {
		if id, err := parseUint(fid); err == nil { folderID = &id }
	}
	files, err := h.repo.ListByFolder(uid, folderID)
	if err != nil { return utils.Respond(c, utils.NewError(500, "list")) }
	return c.JSON(files)
}

func (h *FileHandler) GetRecentFiles(c *fiber.Ctx) error {
	files, err := h.repo.ListRecent(middleware.UserIDFromToken(c), 20)
	if err != nil { return utils.Respond(c, utils.NewError(500, "list recent")) }
	return c.JSON(files)
}

func (h *FileHandler) GetStarredFiles(c *fiber.Ctx) error {
	files, err := h.repo.ListStarred(middleware.UserIDFromToken(c))
	if err != nil { return utils.Respond(c, utils.NewError(500, "list starred")) }
	return c.JSON(files)
}

func (h *FileHandler) GetTrashedFiles(c *fiber.Ctx) error {
	files, err := h.repo.ListTrashed(middleware.UserIDFromToken(c))
	if err != nil { return utils.Respond(c, utils.NewError(500, "list trash")) }
	return c.JSON(files)
}

// DownloadFile returns a short-lived download URL for the file.
// For S3 files: returns a presigned URL the browser can fetch directly.
// For local files: returns a /api/files/:id/stream URL the server will serve.
// The frontend calls this via fetch (with Authorization header), then opens the URL.
func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }

	if file.Status == "processing" {
		return utils.Respond(c, utils.NewError(fiber.StatusConflict, "file is still uploading to cloud storage, try again shortly"))
	}

	if h.s3 != nil && file.FilePath != "" {
		// Generate a 15-minute presigned S3 URL and return it as JSON.
		// The browser fetches this URL directly from S3 — no server traffic.
		url, err := h.s3.PresignDownload(c.Context(), file.FilePath, file.FileName, 15*time.Minute)
		if err != nil {
			slog.Error("presign download failed", "file_id", file.ID, "err", err)
			return utils.Respond(c, utils.NewError(500, "failed to generate download link"))
		}
		slog.Info("download link issued (S3)", "file_id", file.ID, "file_name", file.FileName)
		return c.JSON(fiber.Map{"url": url, "file_name": file.FileName})
	}

	// Local storage — stream the file directly
	if file.FilePath == "" {
		return utils.Respond(c, utils.NewError(404, "file not found on disk"))
	}
	slog.Info("download served (local)", "file_id", file.ID, "file_name", file.FileName)
	c.Set("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	return c.SendFile(file.FilePath)
}

func (h *FileHandler) MoveFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }
	var req struct{ FolderID *uint `json:"folder_id"` }
	if err := c.BodyParser(&req); err != nil { return utils.Respond(c, utils.NewError(400, "bad body")) }
	if err := h.repo.UpdateFolderID(file.ID, req.FolderID); err != nil { return utils.Respond(c, utils.NewError(500, "update")) }
	file.FolderID = req.FolderID
	return c.JSON(file)
}

func (h *FileHandler) ToggleStar(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }
	file.Starred = !file.Starred
	if err := h.repo.Update(file); err != nil { return utils.Respond(c, utils.NewError(500, "update")) }
	return c.JSON(file)
}

func (h *FileHandler) TrashFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }
	if err := h.repo.UpdateTrashed(file.ID, true); err != nil { return utils.Respond(c, utils.NewError(500, "update")) }
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FileHandler) RestoreFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }
	if err := h.repo.UpdateTrashed(file.ID, false); err != nil { return utils.Respond(c, utils.NewError(500, "update")) }
	file.Trashed = false
	return c.JSON(file)
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }

	if h.s3 != nil && file.FilePath != "" {
		if err := h.s3.Delete(c.Context(), file.FilePath); err != nil {
			slog.Warn("s3 delete failed — continuing with DB delete",
				"file_id", file.ID,
				"key",     file.FilePath,
				"err",     err,
			)
		}
	} else if file.FilePath != "" {
		os.Remove(file.FilePath)
	}

	if err := h.repo.Delete(file.ID, middleware.UserIDFromToken(c)); err != nil {
		return utils.Respond(c, utils.NewError(500, "delete"))
	}
	slog.Info("file deleted", "file_id", file.ID, "file_name", file.FileName)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *FileHandler) VerifyChunks(c *fiber.Ctx) error {
	id, err := parseUint(c.Params("id"))
	if err != nil { return utils.Respond(c, utils.NewError(400, "invalid id")) }
	idx, err := h.repo.GetVerifiedChunkIndices(id)
	if err != nil { return utils.Respond(c, utils.NewError(500, "verify")) }
	return c.JSON(fiber.Map{"uploaded_chunks": idx, "total": len(idx)})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint(v), err
}

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "time": time.Now().Unix()})
}