package repository

import (
	"file-transfer-backend/models"
	"file-transfer-backend/types"

	"gorm.io/gorm"
)

type FolderRepository struct{ db *gorm.DB }

func NewFolderRepository(db *gorm.DB) types.IFolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) Create(f *models.Folder) error {
	return r.db.Create(f).Error
}

func (r *FolderRepository) GetByID(id, userID uint) (*models.Folder, error) {
	var f models.Folder
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&f).Error
	return &f, err
}

func (r *FolderRepository) ListByParent(userID uint, parentID *uint) ([]models.Folder, error) {
	var folders []models.Folder
	q := r.db.Where("user_id = ? AND trashed = false", userID)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	err := q.Order("created_at DESC").Find(&folders).Error
	return folders, err
}

func (r *FolderRepository) ListTrashed(userID uint) ([]models.Folder, error) {
	var folders []models.Folder
	err := r.db.Where("user_id = ? AND trashed = true", userID).
		Order("updated_at DESC").
		Find(&folders).Error
	return folders, err
}

func (r *FolderRepository) UpdateTrashed(id, userID uint, trashed bool) error {
	return r.db.Exec(
		"UPDATE folders SET trashed = ? WHERE id = ? AND user_id = ?",
		trashed, id, userID,
	).Error
}

// Delete permanently removes a folder and all files inside it.
// The FK constraint "fk_folders_files" on file_uploads.folder_id requires we
// delete (or nullify) child file_uploads rows before deleting the folder row.
func (r *FolderRepository) Delete(id, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete chunks that belong to files in this folder
		if err := tx.Exec(`
			DELETE FROM file_chunks
			WHERE file_upload_id IN (
				SELECT id FROM file_uploads WHERE folder_id = ?
			)`, id).Error; err != nil {
			return err
		}

		// 2. Delete files inside the folder
		if err := tx.Exec(
			"DELETE FROM file_uploads WHERE folder_id = ?", id,
		).Error; err != nil {
			return err
		}

		// 3. Delete the folder itself (owned by this user)
		return tx.Exec(
			"DELETE FROM folders WHERE id = ? AND user_id = ?", id, userID,
		).Error
	})
}