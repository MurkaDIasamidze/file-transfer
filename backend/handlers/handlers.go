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
	s3   *services.S3Service // nil = локальное хранилище
}

func NewUploadWSHandler(
	repo types.IFileRepository,
	cs   types.IChecksumService,
	cfg  *config.UploadConfig,
	s3   *services.S3Service, // передай nil если S3 не настроен
) *UploadWSHandler {
	return &UploadWSHandler{repo: repo, cs: cs, cfg: cfg, s3: s3}
}

func (h *UploadWSHandler) HandleUpload(conn *websocket.Conn) {
	uid := middleware.WSUserID(conn.Locals)
	defer conn.Close()

	chunks := map[uint]map[int][]byte{} // fileID → chunkIdx → bytes
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
	if err := json.Unmarshal(data, &req); err != nil { wsError(conn, "bad init"); return }

	fu := &models.FileUpload{
		UserID: uid, FolderID: req.FolderID,
		FileName: req.FileName, FileType: req.FileType,
		FileSize: req.FileSize, Checksum: req.Checksum,
		Status: "pending", RelPath: req.RelPath,
	}
	if err := h.repo.Create(fu); err != nil { wsError(conn, "init failed"); return }

	chunks[fu.ID] = map[int][]byte{}
	conn.WriteJSON(map[string]any{"type": "init_ack", "file_upload_id": fu.ID, "file_name": fu.FileName})
}

func (h *UploadWSHandler) wsChunk(conn *websocket.Conn, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req chunkMsg
	if err := json.Unmarshal(data, &req); err != nil { wsError(conn, "bad chunk"); return }

	raw, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil { wsError(conn, "bad base64"); return }

	if sum := sha256hex(raw); sum != req.Checksum {
		wsError(conn, fmt.Sprintf("checksum mismatch chunk %d", req.ChunkIndex)); return
	}

	fc, ok := chunks[req.FileUploadID]
	if !ok { wsError(conn, "unknown file_upload_id"); return }

	fc[req.ChunkIndex] = raw
	totals[req.FileUploadID] = req.TotalChunks

	if len(fc) == 1 {
		if fu, err := h.repo.GetByID(req.FileUploadID); err == nil {
			fu.Status = "uploading"
			fu.TotalChunks = req.TotalChunks
			h.repo.Update(fu)
		}
	}

	conn.WriteJSON(progressMsg{
		Type: "progress", FileUploadID: req.FileUploadID,
		Uploaded: len(fc), Total: req.TotalChunks,
		Percent: float64(len(fc)) / float64(req.TotalChunks) * 100, Status: "uploading",
	})
}

func (h *UploadWSHandler) wsComplete(conn *websocket.Conn, uid uint, data json.RawMessage, chunks map[uint]map[int][]byte, totals map[uint]int) {
	var req completeMsg
	if err := json.Unmarshal(data, &req); err != nil { wsError(conn, "bad complete"); return }

	fu, err := h.repo.GetByID(req.FileUploadID)
	if err != nil || fu.UserID != uid { wsError(conn, "not found or forbidden"); return }

	fc, ok := chunks[req.FileUploadID]
	if !ok || len(fc) == 0 { wsError(conn, "no chunks"); return }

	total := totals[req.FileUploadID]
	if total == 0 { total = len(fc) }

	// ── Склеиваем куски в один буфер ─────────────────────────────────────────
	var buf bytes.Buffer
	for i := range total {
		chunk, ok := fc[i]
		if !ok { wsError(conn, fmt.Sprintf("missing chunk %d", i)); return }
		buf.Write(chunk)
	}
	fileBytes := buf.Bytes()

	// ── Проверяем контрольную сумму всего файла ───────────────────────────────
	if sha256hex(fileBytes) != fu.Checksum {
		fu.Status = "failed"; h.repo.Update(fu)
		wsError(conn, "file checksum mismatch"); return
	}

	// ── Определяем куда сохранять: S3 или локально ───────────────────────────
	if h.s3 != nil {
		// ── S3 путь ───────────────────────────────────────────────────────────
		//
		// Строим ключ S3: "users/42/folders/7/docs/report.pdf"
		var folderID uint
		if fu.FolderID != nil { folderID = *fu.FolderID }

		relPath := fu.RelPath
		if relPath == "" { relPath = fu.FileName }

		s3Key := services.BuildKey(uid, folderID, relPath)

		// Загружаем файл в S3
		if _, err := h.s3.Upload(context.Background(), s3Key, fileBytes, fu.FileType); err != nil {
			fu.Status = "failed"; h.repo.Update(fu)
			wsError(conn, "s3 upload failed: "+err.Error()); return
		}

		// Сохраняем S3 ключ вместо локального пути
		fu.Status   = "completed"
		fu.FilePath = s3Key // теперь FilePath хранит S3 ключ, не путь на диске
		slog.Info("uploaded to S3", "key", s3Key, "file", fu.ID)

	} else {
		// ── Локальный путь (для разработки без AWS) ───────────────────────────
		dir := filepath.Join(h.cfg.Directory, fmt.Sprint(uid))
		if fu.FolderID != nil { dir = filepath.Join(dir, fmt.Sprint(*fu.FolderID)) }
		outPath := filepath.Join(dir, filepath.FromSlash(fu.RelPath))
		if fu.RelPath == "" { outPath = filepath.Join(dir, fu.FileName) }

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			wsError(conn, "mkdir failed"); return
		}
		if err := os.WriteFile(outPath, fileBytes, 0o644); err != nil {
			wsError(conn, "write failed"); return
		}

		fu.Status   = "completed"
		fu.FilePath = outPath
		slog.Info("uploaded locally", "path", outPath, "file", fu.ID)
	}

	h.repo.Update(fu)
	delete(chunks, req.FileUploadID)
	conn.WriteJSON(map[string]any{"type": "done", "file": fu})
}

func wsError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(map[string]any{"type": "error", "message": msg})
}

func sha256hex(b []byte) string {
	s := sha256.Sum256(b); return hex.EncodeToString(s[:])
}

// ─── REST file handler ────────────────────────────────────────────────────────

type FileHandler struct {
	repo types.IFileRepository
	cfg  *config.UploadConfig
	s3   *services.S3Service // nil = локальное хранилище
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

// DownloadFile генерирует временную ссылку для скачивания файла.
// Если S3 включён — возвращает presigned URL (браузер скачивает напрямую из S3).
// Если локально — отдаёт файл через сервер.
func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }

	if h.s3 != nil && file.FilePath != "" {
		// Генерируем временную ссылку на 15 минут
		url, err := h.s3.PresignDownload(c.Context(), file.FilePath, 15*time.Minute)
		if err != nil { return utils.Respond(c, utils.NewError(500, "failed to generate download link")) }
		// Редиректим браузер напрямую на S3 — сервер не тратит трафик
		return c.Redirect(url, fiber.StatusTemporaryRedirect)
	}

	// Локальное скачивание
	if file.FilePath == "" {
		return utils.Respond(c, utils.NewError(404, "file not found on disk"))
	}
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

// DeleteFile удаляет файл из базы данных и из S3 (или с диска).
func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	file, err := h.fileOwner(c)
	if err != nil { return err }

	// Удаляем из S3 если включён
	if h.s3 != nil && file.FilePath != "" {
		if err := h.s3.Delete(c.Context(), file.FilePath); err != nil {
			slog.Warn("s3 delete failed", "key", file.FilePath, "err", err)
			// Не останавливаем — удаляем запись из БД в любом случае
		}
	} else if file.FilePath != "" {
		// Удаляем локальный файл
		os.Remove(file.FilePath)
	}

	if err := h.repo.Delete(file.ID, middleware.UserIDFromToken(c)); err != nil {
		return utils.Respond(c, utils.NewError(500, "delete"))
	}
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