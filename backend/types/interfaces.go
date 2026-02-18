package types

import (
	"file-transfer-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ── Handlers ──────────────────────────────────────────────
type IAuthHandler interface {
	Register(c *fiber.Ctx) error
	Login(c *fiber.Ctx) error
	Me(c *fiber.Ctx) error
}

type IFileHandler interface {
	InitUpload(c *fiber.Ctx) error
	UploadChunk(c *fiber.Ctx) error
	CompleteUpload(c *fiber.Ctx) error
	VerifyChunks(c *fiber.Ctx) error
	ListFiles(c *fiber.Ctx) error
	GetRecentFiles(c *fiber.Ctx) error
	GetStarredFiles(c *fiber.Ctx) error
	GetTrashedFiles(c *fiber.Ctx) error
	MoveFile(c *fiber.Ctx) error
	ToggleStar(c *fiber.Ctx) error
	TrashFile(c *fiber.Ctx) error
	RestoreFile(c *fiber.Ctx) error
	DeleteFile(c *fiber.Ctx) error
	HandleWebSocket(c *websocket.Conn)
}

type IFolderHandler interface {
	CreateFolder(c *fiber.Ctx) error
	ListFolders(c *fiber.Ctx) error
	GetTrashedFolders(c *fiber.Ctx) error
	TrashFolder(c *fiber.Ctx) error
	RestoreFolder(c *fiber.Ctx) error
	DeleteFolder(c *fiber.Ctx) error
}

// ── Repositories ──────────────────────────────────────────
type IUserRepository interface {
	Create(u *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
}

type IFileRepository interface {
	Create(f *models.FileUpload) error
	GetByID(id uint) (*models.FileUpload, error)
	Update(f *models.FileUpload) error
	UpdateFolderID(id uint, folderID *uint) error
	UpdateTrashed(id uint, trashed bool) error
	Delete(id, userID uint) error
	ListByFolder(userID uint, folderID *uint) ([]models.FileUpload, error)
	ListRecent(userID uint, limit int) ([]models.FileUpload, error)
	ListStarred(userID uint) ([]models.FileUpload, error)
	ListTrashed(userID uint) ([]models.FileUpload, error)
	CreateChunk(ch *models.FileChunk) error
	GetChunk(fileID uint, index int) (*models.FileChunk, error)
	UpdateChunk(ch *models.FileChunk) error
	GetChunksByFileID(fileID uint) ([]models.FileChunk, error)
	GetVerifiedChunkIndices(fileID uint) ([]int, error)
}

type IFolderRepository interface {
	Create(f *models.Folder) error
	GetByID(id, userID uint) (*models.Folder, error)
	ListByParent(userID uint, parentID *uint) ([]models.Folder, error)
	ListTrashed(userID uint) ([]models.Folder, error)
	UpdateTrashed(id, userID uint, trashed bool) error
	Delete(id, userID uint) error
}

// ── Services ──────────────────────────────────────────────
type IAuthService interface {
	Register(name, email, password string) (*models.User, error)
	Login(email, password string) (string, *models.User, error)
	Me(id uint) (*models.User, error)
}

type IChecksumService interface {
	Calculate(data []byte) string
	Verify(data []byte, expected string) bool
}

type IFileService interface {
	Reconstruct(fu *models.FileUpload, chunks []models.FileChunk, path string) error
	VerifyFile(path, checksum string) (bool, error)
}