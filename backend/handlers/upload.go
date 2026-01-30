package handlers

import (
	"file-transfer-backend/config"
	"file-transfer-backend/database"
	"file-transfer-backend/models"
	"file-transfer-backend/repository"
	"file-transfer-backend/services"
	"file-transfer-backend/types"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type UploadHandler struct {
	repo            types.IFileRepository
	checksumService types.IChecksumService
	fileService     types.IFileService
	config          *config.UploadConfig
	wsClients       map[uint]map[*websocket.Conn]bool
	wsMutex         sync.RWMutex
}

func NewUploadHandler(db database.IDatabase, cfg *config.UploadConfig) types.IUploadHandler {
	checksumService := services.NewChecksumService()
	return &UploadHandler{
		repo:            repository.NewFileRepository(db.GetDB()),
		checksumService: checksumService,
		fileService:     services.NewFileService(checksumService),
		config:          cfg,
		wsClients:       make(map[uint]map[*websocket.Conn]bool),
	}
}

type InitUploadRequest struct {
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	FileSize    int64  `json:"file_size"`
	TotalChunks int    `json:"total_chunks"`
	Checksum    string `json:"checksum"`
}

type CompleteUploadRequest struct {
	FileUploadID uint `json:"file_upload_id"`
}

type UploadStatusResponse struct {
	ID              uint     `json:"id"`
	FileName        string   `json:"file_name"`
	Status          string   `json:"status"`
	UploadedChunks  []int    `json:"uploaded_chunks"`
	TotalChunks     int      `json:"total_chunks"`
	UploadedCount   int      `json:"uploaded_count"`
	ProgressPercent float64  `json:"progress_percent"`
}

func (h *UploadHandler) InitUpload(c *fiber.Ctx) error {
	var req InitUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	fileUpload := &models.FileUpload{
		FileName:    req.FileName,
		FileType:    req.FileType,
		FileSize:    req.FileSize,
		TotalChunks: req.TotalChunks,
		Checksum:    req.Checksum,
		Status:      "pending",
	}

	if err := h.repo.CreateFileUpload(fileUpload); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize upload"})
	}

	return c.JSON(fiber.Map{
		"file_upload_id": fileUpload.ID,
		"message":        "Upload initialized",
	})
}

func (h *UploadHandler) UploadChunk(c *fiber.Ctx) error {
	fileUploadID, err := strconv.ParseUint(c.FormValue("file_upload_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file upload ID"})
	}

	chunkIndex, err := strconv.Atoi(c.FormValue("chunk_index"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid chunk index"})
	}

	checksum := c.FormValue("checksum")

	file, err := c.FormFile("chunk")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No chunk data provided"})
	}

	fileData, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read chunk"})
	}
	defer fileData.Close()

	data := make([]byte, file.Size)
	if _, err = fileData.Read(data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read chunk data"})
	}

	if !h.checksumService.Verify(data, checksum) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Checksum verification failed"})
	}

	existingChunk, err := h.repo.GetChunk(uint(fileUploadID), chunkIndex)
	if err == nil {
		existingChunk.Data = data
		existingChunk.Checksum = checksum
		existingChunk.ChunkSize = len(data)
		existingChunk.Status = "verified"
		h.repo.UpdateChunk(existingChunk)
	} else {
		chunk := &models.FileChunk{
			FileUploadID: uint(fileUploadID),
			ChunkIndex:   chunkIndex,
			ChunkSize:    len(data),
			Checksum:     checksum,
			Status:       "verified",
			Data:         data,
		}

		if err := h.repo.CreateChunk(chunk); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save chunk"})
		}
	}

	fileUpload, _ := h.repo.GetFileUpload(uint(fileUploadID))
	if fileUpload != nil {
		fileUpload.Status = "uploading"
		h.repo.UpdateFileUpload(fileUpload)
	}

	h.broadcastProgress(uint(fileUploadID))

	return c.JSON(fiber.Map{
		"message":     "Chunk uploaded successfully",
		"chunk_index": chunkIndex,
	})
}

func (h *UploadHandler) CompleteUpload(c *fiber.Ctx) error {
	var req CompleteUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	fileUpload, err := h.repo.GetFileUpload(req.FileUploadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File upload not found"})
	}

	chunks, err := h.repo.GetChunksByFileID(req.FileUploadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch chunks"})
	}

	if len(chunks) != fileUpload.TotalChunks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           "Not all chunks uploaded",
			"uploaded_chunks": len(chunks),
			"total_chunks":    fileUpload.TotalChunks,
		})
	}

	filePath := filepath.Join(h.config.Directory, fileUpload.FileName)

	if err := h.fileService.ReconstructFile(fileUpload, chunks, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	verified, err := h.fileService.VerifyCompleteFile(filePath, fileUpload.Checksum)
	if err != nil || !verified {
		fileUpload.Status = "failed"
		h.repo.UpdateFileUpload(fileUpload)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Final checksum verification failed"})
	}

	fileUpload.Status = "completed"
	fileUpload.FilePath = filePath
	h.repo.UpdateFileUpload(fileUpload)

	h.broadcastProgress(req.FileUploadID)

	return c.JSON(fiber.Map{
		"message":   "File upload completed successfully",
		"file_path": filePath,
		"file_name": fileUpload.FileName,
	})
}

func (h *UploadHandler) VerifyChunk(c *fiber.Ctx) error {
	fileUploadID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file upload ID"})
	}

	indices, err := h.repo.GetUploadedChunkIndices(uint(fileUploadID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch chunks"})
	}

	return c.JSON(fiber.Map{
		"uploaded_chunks": indices,
		"total_uploaded":  len(indices),
	})
}

func (h *UploadHandler) GetUploadStatus(c *fiber.Ctx) error {
	fileUploadID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid file upload ID"})
	}

	fileUpload, err := h.repo.GetFileUpload(uint(fileUploadID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File upload not found"})
	}

	indices, _ := h.repo.GetUploadedChunkIndices(uint(fileUploadID))
	
	progress := float64(0)
	if fileUpload.TotalChunks > 0 {
		progress = (float64(len(indices)) / float64(fileUpload.TotalChunks)) * 100
	}

	return c.JSON(UploadStatusResponse{
		ID:              fileUpload.ID,
		FileName:        fileUpload.FileName,
		Status:          fileUpload.Status,
		UploadedChunks:  indices,
		TotalChunks:     fileUpload.TotalChunks,
		UploadedCount:   len(indices),
		ProgressPercent: progress,
	})
}

func (h *UploadHandler) HandleWebSocket(c *websocket.Conn) {
	fileUploadIDStr := c.Params("id")
	fileUploadID, err := strconv.ParseUint(fileUploadIDStr, 10, 32)
	if err != nil {
		c.Close()
		return
	}

	h.wsMutex.Lock()
	if h.wsClients[uint(fileUploadID)] == nil {
		h.wsClients[uint(fileUploadID)] = make(map[*websocket.Conn]bool)
	}
	h.wsClients[uint(fileUploadID)][c] = true
	h.wsMutex.Unlock()

	defer func() {
		h.wsMutex.Lock()
		delete(h.wsClients[uint(fileUploadID)], c)
		h.wsMutex.Unlock()
		c.Close()
	}()

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

func (h *UploadHandler) broadcastProgress(fileUploadID uint) {
	h.wsMutex.RLock()
	clients := h.wsClients[fileUploadID]
	h.wsMutex.RUnlock()

	if clients == nil {
		return
	}

	indices, _ := h.repo.GetUploadedChunkIndices(fileUploadID)
	fileUpload, _ := h.repo.GetFileUpload(fileUploadID)

	progress := float64(0)
	if fileUpload != nil && fileUpload.TotalChunks > 0 {
		progress = (float64(len(indices)) / float64(fileUpload.TotalChunks)) * 100
	}

	message := fiber.Map{
		"type":             "progress",
		"uploaded_chunks":  len(indices),
		"total_chunks":     fileUpload.TotalChunks,
		"progress_percent": progress,
		"status":           fileUpload.Status,
	}

	for client := range clients {
		if err := client.WriteJSON(message); err != nil {
			client.Close()
			h.wsMutex.Lock()
			delete(h.wsClients[fileUploadID], client)
			h.wsMutex.Unlock()
		}
	}
}