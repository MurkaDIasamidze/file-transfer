package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey"           json:"id"`
	Name      string         `gorm:"not null"             json:"name"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null"             json:"-"`
	CreatedAt time.Time      `                            json:"created_at"`
	UpdatedAt time.Time      `                            json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                json:"-"`
	Folders   []Folder       `gorm:"foreignKey:UserID"    json:"folders,omitempty"`
	Files     []FileUpload   `gorm:"foreignKey:UserID"    json:"files,omitempty"`
}

type Folder struct {
	ID        uint           `gorm:"primarykey"          json:"id"`
	UserID    uint           `gorm:"not null;index"      json:"user_id"`
	ParentID  *uint          `gorm:"index"               json:"parent_id"`
	Name      string         `gorm:"not null"            json:"name"`
	Trashed   bool           `gorm:"default:false"       json:"trashed"`
	CreatedAt time.Time      `                           json:"created_at"`
	UpdatedAt time.Time      `                           json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"               json:"-"`
	Children  []Folder       `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Files     []FileUpload   `gorm:"foreignKey:FolderID" json:"files,omitempty"`
}

type FileUpload struct {
	ID          uint           `gorm:"primarykey"              json:"id"`
	UserID      uint           `gorm:"not null;index"          json:"user_id"`
	FolderID    *uint          `gorm:"index"                   json:"folder_id"`
	FileName    string         `gorm:"not null"                json:"file_name"`
	FileType    string         `                               json:"file_type"`
	FileSize    int64          `                               json:"file_size"`
	TotalChunks int            `                               json:"total_chunks"`
	Checksum    string         `                               json:"checksum"`
	Status      string         `gorm:"default:'pending'"       json:"status"`
	FilePath    string         `                               json:"file_path"`
	Starred     bool           `gorm:"default:false"           json:"starred"`
	Trashed     bool           `gorm:"default:false"           json:"trashed"`
	CreatedAt   time.Time      `                               json:"created_at"`
	UpdatedAt   time.Time      `                               json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"                   json:"-"`
	Chunks      []FileChunk    `gorm:"foreignKey:FileUploadID" json:"-"`
}

type FileChunk struct {
	ID           uint           `gorm:"primarykey"        json:"id"`
	FileUploadID uint           `gorm:"not null;index"    json:"file_upload_id"`
	ChunkIndex   int            `                         json:"chunk_index"`
	ChunkSize    int            `                         json:"chunk_size"`
	Checksum     string         `                         json:"checksum"`
	Status       string         `gorm:"default:'pending'" json:"status"`
	Data         []byte         `                         json:"-"`
	CreatedAt    time.Time      `                         json:"created_at"`
	UpdatedAt    time.Time      `                         json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"             json:"-"`
}