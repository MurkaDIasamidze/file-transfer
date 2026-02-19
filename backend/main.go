package main

import (
	"file-transfer-backend/config"
	"file-transfer-backend/database"
	"file-transfer-backend/handlers"
	"file-transfer-backend/middleware"
	"file-transfer-backend/repository"
	"file-transfer-backend/services"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using system env")
	}

	cfg := config.LoadConfig()

	// ── Database ──────────────────────────────────────────
	db := database.New(&cfg.Database)
	if err := db.Connect(); err != nil {
		slog.Error("db connect", "err", err)
		if os.Getenv("DROP_TABLES") == "true" {
			slog.Warn("resetting database")
			rawDB := db.GetDB()
			for _, t := range []string{"file_chunks", "file_uploads", "folders", "users"} {
				rawDB.Exec("DROP TABLE IF EXISTS " + t + " CASCADE")
			}
			db = database.New(&cfg.Database)
			if err := db.Connect(); err != nil {
				slog.Error("db reconnect failed", "err", err)
				os.Exit(1)
			}
		} else {
			os.Exit(1)
		}
	}
	defer db.Close()

	if err := os.MkdirAll(cfg.Upload.Directory, os.ModePerm); err != nil {
		slog.Error("mkdir uploads", "err", err)
		os.Exit(1)
	}

	// ── Wire ──────────────────────────────────────────────
	gdb := db.GetDB()

	userRepo   := repository.NewUserRepository(gdb)
	fileRepo   := repository.NewFileRepository(gdb)
	folderRepo := repository.NewFolderRepository(gdb)

	cs      := services.NewChecksumService()
	fileSvc := services.NewFileService(cs)
	authSvc := services.NewAuthService(userRepo, &cfg.JWT)

	authHandler     := handlers.NewAuthHandler(authSvc, userRepo)
	fileHandler     := handlers.NewFileHandler(fileRepo, cs, fileSvc, &cfg.Upload)
	folderHandler   := handlers.NewFolderHandler(folderRepo)
	uploadWSHandler := handlers.NewUploadWSHandler(fileRepo, cs, fileSvc, &cfg.Upload)

	// ── Fiber ─────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		BodyLimit: cfg.Server.MaxBodySize,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			slog.Error("unhandled", "path", c.Path(), "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		},
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(middleware.Logger())

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// ── Public routes ─────────────────────────────────────
	app.Get("/health", handlers.HealthCheck)

	auth := app.Group("/api/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login",    authHandler.Login)

	// ── Protected REST routes ─────────────────────────────
	api := app.Group("/api", middleware.JWTMiddleware(&cfg.JWT))
	api.Get("/me",           authHandler.Me)
	api.Patch("/me",         authHandler.UpdateProfile)
	api.Post("/me/password", authHandler.ChangePassword)

	// Files
	api.Get("/files",               fileHandler.ListFiles)
	api.Get("/files/recent",        fileHandler.GetRecentFiles)
	api.Get("/files/starred",       fileHandler.GetStarredFiles)
	api.Get("/files/trash",         fileHandler.GetTrashedFiles)
	api.Patch("/files/:id/move",    fileHandler.MoveFile)
	api.Patch("/files/:id/star",    fileHandler.ToggleStar)
	api.Patch("/files/:id/trash",   fileHandler.TrashFile)
	api.Patch("/files/:id/restore", fileHandler.RestoreFile)
	api.Delete("/files/:id",        fileHandler.DeleteFile)

	// Folders
	api.Post("/folders",                 folderHandler.CreateFolder)
	api.Get("/folders",                  folderHandler.ListFolders)
	api.Get("/folders/trash",            folderHandler.GetTrashedFolders)
	api.Patch("/folders/:id/trash",      folderHandler.TrashFolder)
	api.Patch("/folders/:id/restore",    folderHandler.RestoreFolder)
	api.Delete("/folders/:id",           folderHandler.DeleteFolder)

	// ── WebSocket upload ──────────────────────────────────
	// Token is validated by a lightweight WS-aware JWT check before upgrade.
	// We store the fiber.Ctx in Locals so the handler can call UserIDFromToken.
	app.Get("/ws/upload", middleware.WSJWTMiddleware(&cfg.JWT), websocket.New(func(c *websocket.Conn) {
		// Pass ctx via locals set by WSJWTMiddleware
		uploadWSHandler.HandleUpload(c)
	}))

	slog.Info("server starting", "port", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}