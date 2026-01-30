package repository

import (
	"file-transfer-backend/models"
	"file-transfer-backend/types"

	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) types.IFileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) CreateFileUpload(file *models.FileUpload) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) GetFileUpload(id uint) (*models.FileUpload, error) {
	var file models.FileUpload
	err := r.db.Preload("Chunks").First(&file, id).Error
	return &file, err
}

func (r *FileRepository) UpdateFileUpload(file *models.FileUpload) error {
	return r.db.Save(file).Error
}

func (r *FileRepository) CreateChunk(chunk *models.FileChunk) error {
	return r.db.Create(chunk).Error
}

func (r *FileRepository) GetChunk(fileUploadID uint, chunkIndex int) (*models.FileChunk, error) {
	var chunk models.FileChunk
	err := r.db.Where("file_upload_id = ? AND chunk_index = ?", fileUploadID, chunkIndex).First(&chunk).Error
	return &chunk, err
}

func (r *FileRepository) UpdateChunk(chunk *models.FileChunk) error {
	return r.db.Save(chunk).Error
}

func (r *FileRepository) GetChunksByFileID(fileUploadID uint) ([]models.FileChunk, error) {
	var chunks []models.FileChunk
	err := r.db.Where("file_upload_id = ?", fileUploadID).Order("chunk_index ASC").Find(&chunks).Error
	return chunks, err
}

func (r *FileRepository) GetUploadedChunkIndices(fileUploadID uint) ([]int, error) {
	var chunks []models.FileChunk
	err := r.db.Where("file_upload_id = ? AND status = ?", fileUploadID, "verified").Find(&chunks).Error
	
	indices := make([]int, len(chunks))
	for i, chunk := range chunks {
		indices[i] = chunk.ChunkIndex
	}
	
	return indices, err
}