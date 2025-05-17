package routes

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackson/supabase-go/db"
	"github.com/jackson/supabase-go/middleware"
)

// QueryRequest represents a request to execute a custom SQL query
type QueryRequest struct {
	SQL         string        `json:"sql"`
	Parameters  []interface{} `json:"parameters,omitempty"`
}

// ExecuteQuery handles execution of custom SQL queries
func ExecuteQuery(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		
		// Parse the request
		var req QueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
		}

		// Validate the query
		if req.SQL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SQL query is required",
			})
		}

		// Extract user information for RLS checks
		user := c.Locals("user")
		userID := c.Locals("userId")
		
		// Check if the query is a read-only SELECT query
		isSelect, err := isSelectQuery(req.SQL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid SQL query: %v", err),
			})
		}

		// Apply RLS for write operations (non-SELECT queries)
		if !isSelect {
			allowed, err := middleware.CheckRLS(database, user, "", "write")
			if err != nil || !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access denied by row-level security policy for write operations",
				})
			}
		}

		// Execute the query
		var result interface{}
		var queryErr error

		if isSelect {
			// For SELECT queries, return rows
			rows, err := database.Query(ctx, req.SQL, req.Parameters...)
			if err != nil {
				queryErr = err
			} else {
				defer rows.Close()
				result, queryErr = pgxRowsToJSON(rows)
			}
		} else {
			// For non-SELECT queries, return command tag
			commandTag, err := database.Exec(ctx, req.SQL, req.Parameters...)
			if err != nil {
				queryErr = err
			} else {
				result = fiber.Map{
					"rows_affected": commandTag.RowsAffected(),
					"command":       commandTag.String(),
				}
			}
		}

		if queryErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Query execution failed: %v", queryErr),
			})
		}

		return c.JSON(fiber.Map{
			"data": result,
		})
	}
}

// isSelectQuery checks if the SQL query is a read-only SELECT statement
func isSelectQuery(sql string) (bool, error) {
	// This is a simplified check and could be improved with proper SQL parsing
	trimmedSQL := trimSpaces(sql)
	
	// Check if it starts with SELECT
	if len(trimmedSQL) >= 6 && (trimmedSQL[:6] == "SELECT" || trimmedSQL[:6] == "select") {
		// Naive check for INSERT/UPDATE/DELETE within the query
		// A more robust solution would use proper SQL parsing
		if containsString(trimmedSQL, "INSERT") || 
		   containsString(trimmedSQL, "UPDATE") || 
		   containsString(trimmedSQL, "DELETE") ||
		   containsString(trimmedSQL, "DROP") ||
		   containsString(trimmedSQL, "ALTER") ||
		   containsString(trimmedSQL, "CREATE") {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

// Helper to trim spaces from a string
func trimSpaces(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			result += string(s[i])
		}
	}
	return result
}

// Helper to check if a string contains another string (case insensitive)
func containsString(s, substr string) bool {
	s = strings.ToUpper(s)
	substr = strings.ToUpper(substr)
	return strings.Contains(s, substr)
}
