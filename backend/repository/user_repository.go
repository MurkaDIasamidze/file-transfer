package repository

import (
	"file-transfer-backend/models"
	"file-transfer-backend/types"

	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) types.IUserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *models.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Where("email = ?", email).First(&u).Error
	return &u, err
}

func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var u models.User
	err := r.db.First(&u, id).Error
	return &u, err
}