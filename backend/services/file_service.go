package services

import (
	"crypto/sha256"
	"encoding/hex"
	"file-transfer-backend/models"
	"file-transfer-backend/types"
	"file-transfer-backend/utils"
	"fmt"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
)

// ── ChecksumService ───────────────────────────────────────

type ChecksumService struct{}

func NewChecksumService() types.IChecksumService { return &ChecksumService{} }

func (s *ChecksumService) Calculate(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func (s *ChecksumService) Verify(data []byte, expected string) bool {
	return s.Calculate(data) == expected
}

// ── FileService ───────────────────────────────────────────

type FileService struct {
	cs types.IChecksumService
}

func NewFileService(cs types.IChecksumService) types.IFileService {
	return &FileService{cs: cs}
}

func (s *FileService) Reconstruct(fu *models.FileUpload, chunks []models.FileChunk, path string) error {
	slog.Info("reconstructing file", "name", fu.FileName, "chunks", fu.TotalChunks)

	f, err := os.Create(path)
	if err != nil {
		return utils.NewError(fiber.StatusInternalServerError,
			fmt.Sprintf("create file: %s", err))
	}
	defer f.Close()

	for i := 0; i < fu.TotalChunks; i++ {
		found := false
		for _, ch := range chunks {
			if ch.ChunkIndex == i {
				if _, err := f.Write(ch.Data); err != nil {
					return utils.NewError(fiber.StatusInternalServerError,
						fmt.Sprintf("write chunk %d: %s", i, err))
				}
				found = true
				break
			}
		}
		if !found {
			return utils.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("chunk %d missing", i))
		}
	}

	slog.Info("file reconstructed", "path", path)
	return nil
}

func (s *FileService) VerifyFile(path, checksum string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, utils.NewError(fiber.StatusInternalServerError,
			fmt.Sprintf("read file: %s", err))
	}
	ok := s.cs.Verify(data, checksum)
	slog.Info("file checksum verify", "path", path, "ok", ok)
	return ok, nil
}