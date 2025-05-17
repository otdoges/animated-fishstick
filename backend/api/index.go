package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackson/supabase-go/config"
	"github.com/jackson/supabase-go/db"
	"github.com/jackson/supabase-go/middleware"
	"github.com/jackson/supabase-go/routes"
)

var app *fiber.App
var database *db.DB

// init initializes the application
func init() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	// Initialize database
	database, err = db.Connect(cfg.Database)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}

	// Run migrations
	if err := database.RunMigrations(context.Background()); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
	}

	// Create Fiber app
	app = fiber.New(fiber.Config{
		AppName:               "Supabase Go",
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Authentication middleware
	app.Use(middleware.ClerkAuth(cfg.Auth))

	// Set up routes
	routes.Setup(app, database)
}

// Handler is the Vercel serverless function handler
func Handler(w http.ResponseWriter, r *http.Request) {
	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Adjust path for Vercel
	// Convert /api/v1/users to /v1/users
	path := r.URL.Path
	if strings.HasPrefix(path, "/api") {
		r.URL.Path = strings.TrimPrefix(path, "/api")
	}

	// Handle request using Fiber
	if err := app.Test(r, int(w.Header().Get("Content-Length"))); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error: %v", err)))
	}
}
