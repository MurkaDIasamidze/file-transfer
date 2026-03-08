package repository

import (
	"file-transfer-backend/models"
	"file-transfer-backend/types"

	"gorm.io/gorm"
)

type FileRepository struct{ db *gorm.DB }

func NewFileRepository(db *gorm.DB) types.IFileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(f *models.FileUpload) error {
	return r.db.Create(f).Error
}

func (r *FileRepository) GetByID(id uint) (*models.FileUpload, error) {
	var f models.FileUpload
	// No Preload("Chunks") — chunks are held in-memory during WS upload,
	// not persisted to file_chunks anymore.
	err := r.db.First(&f, id).Error
	return &f, err
}

func (r *FileRepository) Update(f *models.FileUpload) error {
	return r.db.Save(f).Error
}

func (r *FileRepository) UpdateFolderID(id uint, folderID *uint) error {
	return r.db.Exec("UPDATE file_uploads SET folder_id = ? WHERE id = ?", folderID, id).Error
}

func (r *FileRepository) UpdateTrashed(id uint, trashed bool) error {
	return r.db.Exec("UPDATE file_uploads SET trashed = ? WHERE id = ?", trashed, id).Error
}

func (r *FileRepository) Delete(id, userID uint) error {
	if err := r.db.Exec("DELETE FROM file_chunks WHERE file_upload_id = ?", id).Error; err != nil {
		return err
	}
	return r.db.Exec("DELETE FROM file_uploads WHERE id = ? AND user_id = ?", id, userID).Error
}

func (r *FileRepository) ListByFolder(userID uint, folderID *uint) ([]models.FileUpload, error) {
	var files []models.FileUpload
	q := r.db.Where("user_id = ? AND status = 'completed' AND trashed = false", userID)
	if folderID == nil {
		q = q.Where("folder_id IS NULL")
	} else {
		q = q.Where("folder_id = ?", *folderID)
	}
	err := q.Order("created_at DESC").Find(&files).Error
	return files, err
}

func (r *FileRepository) ListRecent(userID uint, limit int) ([]models.FileUpload, error) {
	var files []models.FileUpload
	err := r.db.Where("user_id = ? AND status = 'completed' AND trashed = false", userID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&files).Error
	return files, err
}

func (r *FileRepository) ListStarred(userID uint) ([]models.FileUpload, error) {
	var files []models.FileUpload
	err := r.db.Where("user_id = ? AND starred = true AND trashed = false", userID).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (r *FileRepository) ListTrashed(userID uint) ([]models.FileUpload, error) {
	var files []models.FileUpload
	err := r.db.Where("user_id = ? AND trashed = true", userID).
		Order("updated_at DESC").
		Find(&files).Error
	return files, err
}

// The following chunk methods are kept to satisfy the IFileRepository interface
// and support the /upload/verify/:id endpoint. They are not used by the WS
// upload path (which holds chunks in memory).

func (r *FileRepository) CreateChunk(ch *models.FileChunk) error {
	return r.db.Create(ch).Error
}

func (r *FileRepository) GetChunk(fileID uint, index int) (*models.FileChunk, error) {
	var ch models.FileChunk
	err := r.db.Where("file_upload_id = ? AND chunk_index = ?", fileID, index).First(&ch).Error
	return &ch, err
}

func (r *FileRepository) UpdateChunk(ch *models.FileChunk) error {
	return r.db.Save(ch).Error
}

func (r *FileRepository) GetChunksByFileID(fileID uint) ([]models.FileChunk, error) {
	var chunks []models.FileChunk
	err := r.db.Where("file_upload_id = ?", fileID).
		Order("chunk_index ASC").
		Find(&chunks).Error
	return chunks, err
}

func (r *FileRepository) GetVerifiedChunkIndices(fileID uint) ([]int, error) {
	var chunks []models.FileChunk
	err := r.db.Select("chunk_index").
		Where("file_upload_id = ? AND status = 'verified'", fileID).
		Find(&chunks).Error
	idx := make([]int, len(chunks))
	for i, ch := range chunks {
		idx[i] = ch.ChunkIndex
	}
	return idx, err
}