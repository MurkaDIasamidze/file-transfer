package services

import (
	"crypto/sha256"
	"encoding/hex"
	"file-transfer-backend/models"
	"file-transfer-backend/types"
	"fmt"
	"os"
)

type ChecksumService struct{}

func NewChecksumService() types.IChecksumService {
	return &ChecksumService{}
}

func (s *ChecksumService) Calculate(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *ChecksumService) Verify(data []byte, expectedChecksum string) bool {
	actualChecksum := s.Calculate(data)
	return actualChecksum == expectedChecksum
}

type FileService struct {
	checksumService types.IChecksumService
}

func NewFileService(checksumService types.IChecksumService) types.IFileService {
	return &FileService{
		checksumService: checksumService,
	}
}

func (s *FileService) ReconstructFile(fileUpload *models.FileUpload, chunks []models.FileChunk, outputPath string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	for i := 0; i < fileUpload.TotalChunks; i++ {
		found := false
		for _, chunk := range chunks {
			if chunk.ChunkIndex == i {
				if _, err := outputFile.Write(chunk.Data); err != nil {
					return fmt.Errorf("failed to write chunk %d: %w", i, err)
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("chunk %d not found", i)
		}
	}

	return nil
}

func (s *FileService) VerifyCompleteFile(filePath string, expectedChecksum string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	return s.checksumService.Verify(data, expectedChecksum), nil
}