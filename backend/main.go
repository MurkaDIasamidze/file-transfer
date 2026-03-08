// main.go — точка входа, собирает все зависимости вместе
package main

import (
	"file-transfer-backend/config"
	"file-transfer-backend/database"
	"file-transfer-backend/handlers"
	"file-transfer-backend/middleware"
	"file-transfer-backend/repository"
	"file-transfer-backend/services"
	"log"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Загружаем .env файл
	godotenv.Load()

	// 2. Читаем конфиг из переменных окружения
	cfg := config.LoadConfig()

	// 3. Подключаемся к PostgreSQL и мигрируем схему
	db := database.New(&cfg.Database)
	if err := db.Connect(); err != nil {
		log.Fatalf("database connect: %v", err)
	}
	gdb := db.GetDB()

	// 4. Создаём репозитории (работа с таблицами БД)
	userRepo   := repository.NewUserRepository(gdb)
	fileRepo   := repository.NewFileRepository(gdb)
	folderRepo := repository.NewFolderRepository(gdb)

	// 5. Создаём сервисы (бизнес-логика)
	authSvc := services.NewAuthService(userRepo, &cfg.JWT)
	cs      := services.NewChecksumService()

	// 6. Инициализируем S3 если ключи заданы в .env
	//    Если S3 не настроен — s3Svc будет nil и файлы сохранятся локально.
	var s3Svc *services.S3Service
	if cfg.S3.Enabled {
		var err error
		s3Svc, err = services.NewS3Service(
			cfg.S3.Region,
			cfg.S3.AccessKeyID,
			cfg.S3.SecretAccessKey,
			cfg.S3.Bucket,
		)
		if err != nil {
			log.Fatalf("S3 init failed: %v\nПроверь AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_S3_BUCKET в .env", err)
		}
		slog.Info("S3 storage enabled", "bucket", cfg.S3.Bucket, "region", cfg.S3.Region)
	} else {
		slog.Info("S3 not configured — using local storage", "dir", cfg.Upload.Directory)
	}

	// 7. Создаём HTTP хендлеры (передаём s3Svc — может быть nil)
	authHandler   := handlers.NewAuthHandler(authSvc, userRepo)
	fileHandler   := handlers.NewFileHandler(fileRepo, &cfg.Upload, s3Svc)
	folderHandler := handlers.NewFolderHandler(folderRepo)
	uploadHandler := handlers.NewUploadWSHandler(fileRepo, cs, &cfg.Upload, s3Svc)

	// 8. Создаём Fiber приложение
	app := fiber.New(fiber.Config{
		BodyLimit: cfg.Server.MaxBodySize,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE",
	}))

	// 9. Публичные маршруты (без JWT)
	app.Get("/health", handlers.HealthCheck)
	auth := app.Group("/api/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login",    authHandler.Login)

	// 10. Защищённые маршруты (требуют JWT токен)
	api := app.Group("/api", middleware.JWTMiddleware(&cfg.JWT))

	api.Get("/me",           authHandler.Me)
	api.Patch("/me",         authHandler.UpdateProfile)
	api.Post("/me/password", authHandler.ChangePassword)

	api.Get("/files",                  fileHandler.ListFiles)
	api.Get("/files/recent",           fileHandler.GetRecentFiles)
	api.Get("/files/starred",          fileHandler.GetStarredFiles)
	api.Get("/files/trash",            fileHandler.GetTrashedFiles)
	api.Get("/files/:id/download",     fileHandler.DownloadFile)   // ← новый эндпоинт
	api.Patch("/files/:id/move",       fileHandler.MoveFile)
	api.Patch("/files/:id/star",       fileHandler.ToggleStar)
	api.Patch("/files/:id/trash",      fileHandler.TrashFile)
	api.Patch("/files/:id/restore",    fileHandler.RestoreFile)
	api.Delete("/files/:id",           fileHandler.DeleteFile)

	api.Get("/folders",                folderHandler.ListFolders)
	api.Post("/folders",               folderHandler.CreateFolder)
	api.Get("/folders/trash",          folderHandler.GetTrashedFolders)
	api.Patch("/folders/:id/trash",    folderHandler.TrashFolder)
	api.Patch("/folders/:id/restore",  folderHandler.RestoreFolder)
	api.Delete("/folders/:id",         folderHandler.DeleteFolder)

	// 11. WebSocket загрузка файлов
	app.Use("/ws/upload", middleware.WSJWTMiddleware(&cfg.JWT))
	app.Get("/ws/upload", websocket.New(uploadHandler.HandleUpload))

	// 12. Запускаем сервер
	slog.Info("server starting", "port", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}