package main

import (
	"file-transfer-backend/config"
	"file-transfer-backend/database"
	"file-transfer-backend/handlers"
	"file-transfer-backend/logger"
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
	// 1. Load .env
	godotenv.Load()

	// 2. Set up logger — colored terminal + JSON file
	//    Logs are written to logs/app.json (created automatically)
	slog.SetDefault(logger.New("logs/app.json"))

	// 3. Read config
	cfg := config.LoadConfig()

	// 4. Connect to PostgreSQL
	db := database.New(&cfg.Database)
	if err := db.Connect(); err != nil {
		log.Fatalf("database connect: %v", err)
	}
	gdb := db.GetDB()

	// 5. Repositories
	userRepo   := repository.NewUserRepository(gdb)
	fileRepo   := repository.NewFileRepository(gdb)
	folderRepo := repository.NewFolderRepository(gdb)

	// 6. Services
	authSvc := services.NewAuthService(userRepo, &cfg.JWT)
	cs      := services.NewChecksumService()

	// 7. S3 (optional — enabled only when keys are present in .env)
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
			log.Fatalf("S3 init failed: %v", err)
		}
		slog.Info("S3 async upload enabled",
			"bucket", cfg.S3.Bucket,
			"region", cfg.S3.Region,
		)
	} else {
		slog.Info("S3 not configured — using local storage",
			"dir", cfg.Upload.Directory,
		)
	}

	// 8. Handlers
	authHandler   := handlers.NewAuthHandler(authSvc, userRepo)
	fileHandler   := handlers.NewFileHandler(fileRepo, &cfg.Upload, s3Svc)
	folderHandler := handlers.NewFolderHandler(folderRepo)
	uploadHandler := handlers.NewUploadWSHandler(fileRepo, cs, &cfg.Upload, s3Svc)

	// 9. Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: cfg.Server.MaxBodySize,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE",
	}))

	// 10. Public routes
	app.Get("/health", handlers.HealthCheck)
	auth := app.Group("/api/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login",    authHandler.Login)

	// 11. Protected routes
	api := app.Group("/api", middleware.JWTMiddleware(&cfg.JWT))

	api.Get("/me",           authHandler.Me)
	api.Patch("/me",         authHandler.UpdateProfile)
	api.Post("/me/password", authHandler.ChangePassword)

	api.Get("/files",                fileHandler.ListFiles)
	api.Get("/files/recent",         fileHandler.GetRecentFiles)
	api.Get("/files/starred",        fileHandler.GetStarredFiles)
	api.Get("/files/trash",          fileHandler.GetTrashedFiles)
	api.Get("/files/:id/download",   fileHandler.DownloadFile)
	api.Patch("/files/:id/move",     fileHandler.MoveFile)
	api.Patch("/files/:id/star",     fileHandler.ToggleStar)
	api.Patch("/files/:id/trash",    fileHandler.TrashFile)
	api.Patch("/files/:id/restore",  fileHandler.RestoreFile)
	api.Delete("/files/:id",         fileHandler.DeleteFile)

	api.Get("/folders",               folderHandler.ListFolders)
	api.Post("/folders",              folderHandler.CreateFolder)
	api.Get("/folders/trash",         folderHandler.GetTrashedFolders)
	api.Patch("/folders/:id/trash",   folderHandler.TrashFolder)
	api.Patch("/folders/:id/restore", folderHandler.RestoreFolder)
	api.Delete("/folders/:id",        folderHandler.DeleteFolder)

	// 12. WebSocket upload
	app.Use("/ws/upload", middleware.WSJWTMiddleware(&cfg.JWT))
	app.Get("/ws/upload", websocket.New(uploadHandler.HandleUpload))

	// 13. Start
	slog.Info("server starting", "port", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}