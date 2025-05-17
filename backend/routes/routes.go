package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackson/supabase-go/db"
)

// Setup configures all API routes
func Setup(app *fiber.App, database *db.DB) {
	// API prefix
	api := app.Group("/api")

	// Health check endpoint
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "Service is running",
		})
	})

	// Database schema endpoint
	api.Get("/schema", GetDatabaseSchema(database))

	// Table operations
	tables := api.Group("/tables")
	tables.Get("/", GetAllTables(database))
	tables.Get("/:table", GetTable(database))
	tables.Get("/:table/columns", GetTableColumns(database))
	tables.Get("/:table/rows", GetTableRows(database))
	tables.Post("/:table", CreateTableRow(database))
	tables.Get("/:table/rows/:id", GetTableRowById(database))
	tables.Patch("/:table/rows/:id", UpdateTableRow(database))
	tables.Delete("/:table/rows/:id", DeleteTableRow(database))

	// Query operations
	api.Post("/query", ExecuteQuery(database))

	// Row Level Security policy management
	rls := api.Group("/rls")
	rls.Get("/policies", GetRLSPolicies(database))
	rls.Post("/policies", CreateRLSPolicy(database))
	rls.Get("/policies/:id", GetRLSPolicy(database))
	rls.Patch("/policies/:id", UpdateRLSPolicy(database))
	rls.Delete("/policies/:id", DeleteRLSPolicy(database))
}
