package main

import (
	"file-transfer-backend/config"
	"file-transfer-backend/database"
	"file-transfer-backend/handlers"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db := database.NewDatabase(&cfg.Database)
	if err := db.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create uploads directory
	if err := os.MkdirAll(cfg.Upload.Directory, os.ModePerm); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(db, &cfg.Upload)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit: cfg.Server.MaxBodySize,
	})

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Routes
	api := app.Group("/api")
	
	api.Post("/upload/init", uploadHandler.InitUpload)
	api.Post("/upload/chunk", uploadHandler.UploadChunk)
	api.Post("/upload/complete", uploadHandler.CompleteUpload)
	api.Get("/upload/verify/:id", uploadHandler.VerifyChunk)
	api.Get("/upload/status/:id", uploadHandler.GetUploadStatus)

	// WebSocket route
	app.Get("/ws/upload/:id", websocket.New(uploadHandler.HandleWebSocket))

	// Start server
	log.Printf("Server starting on port %s", cfg.Server.Port)
	log.Fatal(app.Listen(":" + cfg.Server.Port))
}