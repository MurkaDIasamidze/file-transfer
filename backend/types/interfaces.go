package types

import (
	"file-transfer-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type IUploadHandler interface {
	InitUpload(c *fiber.Ctx) error
	UploadChunk(c *fiber.Ctx) error
	CompleteUpload(c *fiber.Ctx) error
	VerifyChunk(c *fiber.Ctx) error
	GetUploadStatus(c *fiber.Ctx) error
	HandleWebSocket(c *websocket.Conn)
}

type IFileRepository interface {
	CreateFileUpload(file *models.FileUpload) error
	GetFileUpload(id uint) (*models.FileUpload, error)
	UpdateFileUpload(file *models.FileUpload) error
	CreateChunk(chunk *models.FileChunk) error
	GetChunk(fileUploadID uint, chunkIndex int) (*models.FileChunk, error)
	UpdateChunk(chunk *models.FileChunk) error
	GetChunksByFileID(fileUploadID uint) ([]models.FileChunk, error)
	GetUploadedChunkIndices(fileUploadID uint) ([]int, error)
}

type IChecksumService interface {
	Calculate(data []byte) string
	Verify(data []byte, expectedChecksum string) bool
}

type IFileService interface {
	ReconstructFile(fileUpload *models.FileUpload, chunks []models.FileChunk, outputPath string) error
	VerifyCompleteFile(filePath string, expectedChecksum string) (bool, error)
}