package handlers

import (
	"file-transfer-backend/database"
	"file-transfer-backend/models"
	"file-transfer-backend/utils"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type InitUploadRequest struct {
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	FileSize    int64  `json:"file_size"`
	TotalChunks int    `json:"total_chunks"`
	Checksum    string `json:"checksum"`
}

type UploadChunkRequest struct {
	FileUploadID uint   `json:"file_upload_id"`
	ChunkIndex   int    `json:"chunk_index"`
	Checksum     string `json:"checksum"`
	Data         []byte `json:"data"`
}

type CompleteUploadRequest struct {
	FileUploadID uint `json:"file_upload_id"`
}

func InitUpload(c *fiber.Ctx) error {
	var req InitUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	fileUpload := models.FileUpload{
		FileName:    req.FileName,
		FileType:    req.FileType,
		FileSize:    req.FileSize,
		TotalChunks: req.TotalChunks,
		Checksum:    req.Checksum,
		Status:      "pending",
	}

	if err := database.DB.Create(&fileUpload).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to initialize upload"})
	}

	return c.JSON(fiber.Map{
		"file_upload_id": fileUpload.ID,
		"message":        "Upload initialized",
	})
}

func UploadChunk(c *fiber.Ctx) error {
	fileUploadID, err := strconv.ParseUint(c.FormValue("file_upload_id"), 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid file upload ID"})
	}

	chunkIndex, err := strconv.Atoi(c.FormValue("chunk_index"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid chunk index"})
	}

	checksum := c.FormValue("checksum")
	
	file, err := c.FormFile("chunk")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No chunk data provided"})
	}

	// Read chunk data
	fileData, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read chunk"})
	}
	defer fileData.Close()

	data := make([]byte, file.Size)
	_, err = fileData.Read(data)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read chunk data"})
	}

	// Verify checksum
	if !utils.VerifyChecksum(data, checksum) {
		return c.Status(400).JSON(fiber.Map{"error": "Checksum verification failed"})
	}

	// Check if chunk already exists
	var existingChunk models.FileChunk
	result := database.DB.Where("file_upload_id = ? AND chunk_index = ?", fileUploadID, chunkIndex).First(&existingChunk)
	
	if result.Error == nil {
		// Update existing chunk
		existingChunk.Data = data
		existingChunk.Checksum = checksum
		existingChunk.ChunkSize = len(data)
		existingChunk.Status = "verified"
		database.DB.Save(&existingChunk)
	} else {
		// Create new chunk
		chunk := models.FileChunk{
			FileUploadID: uint(fileUploadID),
			ChunkIndex:   chunkIndex,
			ChunkSize:    len(data),
			Checksum:     checksum,
			Status:       "verified",
			Data:         data,
		}

		if err := database.DB.Create(&chunk).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save chunk"})
		}
	}

	// Update file upload status
	database.DB.Model(&models.FileUpload{}).Where("id = ?", fileUploadID).Update("status", "uploading")

	return c.JSON(fiber.Map{
		"message":     "Chunk uploaded successfully",
		"chunk_index": chunkIndex,
	})
}

func CompleteUpload(c *fiber.Ctx) error {
	var req CompleteUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	var fileUpload models.FileUpload
	if err := database.DB.Preload("Chunks").First(&fileUpload, req.FileUploadID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "File upload not found"})
	}

	// Verify all chunks are uploaded
	if len(fileUpload.Chunks) != fileUpload.TotalChunks {
		return c.Status(400).JSON(fiber.Map{
			"error":           "Not all chunks uploaded",
			"uploaded_chunks": len(fileUpload.Chunks),
			"total_chunks":    fileUpload.TotalChunks,
		})
	}

	// Reconstruct file
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	filePath := filepath.Join(uploadDir, fileUpload.FileName)
	outputFile, err := os.Create(filePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create output file"})
	}
	defer outputFile.Close()

	// Write chunks in order
	for i := 0; i < fileUpload.TotalChunks; i++ {
		var chunk models.FileChunk
		if err := database.DB.Where("file_upload_id = ? AND chunk_index = ?", fileUpload.ID, i).First(&chunk).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Chunk %d not found", i)})
		}

		if _, err := outputFile.Write(chunk.Data); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to write chunk to file"})
		}
	}

	// Read complete file and verify checksum
	completeData, err := os.ReadFile(filePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read complete file"})
	}

	if !utils.VerifyChecksum(completeData, fileUpload.Checksum) {
		os.Remove(filePath)
		database.DB.Model(&fileUpload).Update("status", "failed")
		return c.Status(400).JSON(fiber.Map{"error": "Final checksum verification failed"})
	}

	// Update file upload status
	fileUpload.Status = "completed"
	fileUpload.FilePath = filePath
	database.DB.Save(&fileUpload)

	return c.JSON(fiber.Map{
		"message":   "File upload completed successfully",
		"file_path": filePath,
		"file_name": fileUpload.FileName,
	})
}

func VerifyChunk(c *fiber.Ctx) error {
	fileUploadID := c.Params("id")

	var chunks []models.FileChunk
	if err := database.DB.Where("file_upload_id = ?", fileUploadID).Find(&chunks).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch chunks"})
	}

	uploadedChunks := make([]int, 0)
	for _, chunk := range chunks {
		if chunk.Status == "verified" {
			uploadedChunks = append(uploadedChunks, chunk.ChunkIndex)
		}
	}

	return c.JSON(fiber.Map{
		"uploaded_chunks": uploadedChunks,
		"total_uploaded":  len(uploadedChunks),
	})
}