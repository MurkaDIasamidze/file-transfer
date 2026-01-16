package models

import (
	"time"

	"gorm.io/gorm"
)

type FileUpload struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	FileName      string         `json:"file_name"`
	FileType      string         `json:"file_type"`
	FileSize      int64          `json:"file_size"`
	TotalChunks   int            `json:"total_chunks"`
	Checksum      string         `json:"checksum"`
	Status        string         `json:"status"` // pending, uploading, completed, failed
	FilePath      string         `json:"file_path"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Chunks        []FileChunk    `gorm:"foreignKey:FileUploadID" json:"chunks,omitempty"`
}

type FileChunk struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	FileUploadID uint           `json:"file_upload_id"`
	ChunkIndex   int            `json:"chunk_index"`
	ChunkSize    int            `json:"chunk_size"`
	Checksum     string         `json:"checksum"`
	Status       string         `json:"status"` // pending, uploaded, verified
	Data         []byte         `json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}