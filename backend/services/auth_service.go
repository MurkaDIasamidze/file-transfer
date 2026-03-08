package services

import (
	"errors"
	"file-transfer-backend/config"
	"file-transfer-backend/models"
	"file-transfer-backend/types"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthService struct {
	repo types.IUserRepository
	cfg  *config.JWTConfig
}

func NewAuthService(repo types.IUserRepository, cfg *config.JWTConfig) types.IAuthService {
	return &AuthService{repo: repo, cfg: cfg}
}

func (s *AuthService) Register(name, email, password string) (*models.User, error) {
	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, errors.New("email already in use")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{Name: name, Email: email, Password: string(hash)}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(email, password string) (string, *models.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return "", nil, errors.New("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("invalid password")
	}
	token, err := s.generateToken(user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) Me(id uint) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *AuthService) generateToken(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Duration(s.cfg.ExpiryHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}