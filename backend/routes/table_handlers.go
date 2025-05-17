package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackson/supabase-go/db"
	"github.com/jackson/supabase-go/middleware"
)

// GetAllTables returns a list of all tables in the database
func GetAllTables(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		
		tables, err := database.GetTables(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get tables: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"tables": tables,
		})
	}
}

// GetTable returns information about a specific table
func GetTable(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		ctx := context.Background()

		// Check if table exists
		tables, err := database.GetTables(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get tables: %v", err),
			})
		}

		tableExists := false
		for _, t := range tables {
			if t == tableName {
				tableExists = true
				break
			}
		}

		if !tableExists {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s' not found", tableName),
			})
		}

		// Get columns for the table
		columns, err := database.GetTableColumns(ctx, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get columns: %v", err),
			})
		}

		// Get row count
		var count int
		err = database.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&count)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get row count: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"name":       tableName,
			"columns":    columns,
			"rowCount":   count,
		})
	}
}

// GetTableColumns returns all columns for a specific table
func GetTableColumns(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		ctx := context.Background()

		// Get columns for the table
		columns, err := database.GetTableColumns(ctx, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get columns: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"columns": columns,
		})
	}
}

// GetTableRows returns rows from a table with filtering and pagination
func GetTableRows(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		ctx := context.Background()

		// Check RLS policies for the current user
		user := c.Locals("user")
		allowed, err := middleware.CheckRLS(database, user, tableName, "select")
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied by row-level security policy",
			})
		}

		// Pagination parameters
		page, _ := strconv.Atoi(c.Query("page", "1"))
		pageSize, _ := strconv.Atoi(c.Query("page_size", "10"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize

		// Build the query with filters
		queryFilters := buildQueryFilters(c)
		
		// Main query
		query := fmt.Sprintf("SELECT * FROM %s", pgx.Identifier{tableName}.Sanitize())
		if len(queryFilters.wheres) > 0 {
			query += " WHERE " + strings.Join(queryFilters.wheres, " AND ")
		}
		
		// Order by
		orderBy := c.Query("order_by")
		orderDir := c.Query("order_dir", "asc")
		if orderBy != "" {
			query += fmt.Sprintf(" ORDER BY %s %s", pgx.Identifier{orderBy}.Sanitize(), orderDir)
		}
		
		// Add pagination
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

		// Execute the query
		rows, err := database.Query(ctx, query, queryFilters.params...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to query table: %v", err),
			})
		}
		defer rows.Close()

		// Convert rows to JSON
		data, err := pgxRowsToJSON(rows)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process results: %v", err),
			})
		}

		// Count total rows (for pagination)
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())
		if len(queryFilters.wheres) > 0 {
			countQuery += " WHERE " + strings.Join(queryFilters.wheres, " AND ")
		}
		
		var total int
		err = database.QueryRow(ctx, countQuery, queryFilters.params...).Scan(&total)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get total count: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"data":       data,
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + pageSize - 1) / pageSize,
		})
	}
}

// GetTableRowById returns a single row by its ID
func GetTableRowById(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		idParam := c.Params("id")
		ctx := context.Background()

		// Check RLS policies for the current user
		user := c.Locals("user")
		allowed, err := middleware.CheckRLS(database, user, tableName, "select")
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied by row-level security policy",
			})
		}

		// Get primary key column
		primaryKeyColumn, err := getPrimaryKeyColumn(ctx, database, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to determine primary key: %v", err),
			})
		}

		// Query the row
		query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", 
			pgx.Identifier{tableName}.Sanitize(), 
			pgx.Identifier{primaryKeyColumn}.Sanitize())
		
		row := database.QueryRow(ctx, query, idParam)
		result, err := pgxRowToJSON(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Row with ID %s not found", idParam),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to query row: %v", err),
			})
		}

		return c.JSON(result)
	}
}

// CreateTableRow creates a new row in the specified table
func CreateTableRow(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		ctx := context.Background()

		// Check RLS policies for the current user
		user := c.Locals("user")
		allowed, err := middleware.CheckRLS(database, user, tableName, "insert")
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied by row-level security policy",
			})
		}

		// Parse request body
		var data map[string]interface{}
		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
		}

		// Get table columns
		columns, err := database.GetTableColumns(ctx, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get columns: %v", err),
			})
		}

		// Build insert query
		columnNames := []string{}
		placeholders := []string{}
		values := []interface{}{}
		paramCounter := 1

		for _, col := range columns {
			if value, exists := data[col.Name]; exists {
				columnNames = append(columnNames, pgx.Identifier{col.Name}.Sanitize())
				placeholders = append(placeholders, fmt.Sprintf("$%d", paramCounter))
				values = append(values, value)
				paramCounter++
			}
		}

		if len(columnNames) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No valid columns provided for insert",
			})
		}

		// Create the INSERT query
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
			pgx.Identifier{tableName}.Sanitize(),
			strings.Join(columnNames, ", "),
			strings.Join(placeholders, ", "),
		)

		// Execute the query
		row := database.QueryRow(ctx, query, values...)
		result, err := pgxRowToJSON(row)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to insert row: %v", err),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(result)
	}
}

// UpdateTableRow updates an existing row in the specified table
func UpdateTableRow(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		idParam := c.Params("id")
		ctx := context.Background()

		// Check RLS policies for the current user
		user := c.Locals("user")
		allowed, err := middleware.CheckRLS(database, user, tableName, "update")
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied by row-level security policy",
			})
		}

		// Parse request body
		var data map[string]interface{}
		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
		}

		// Get primary key column
		primaryKeyColumn, err := getPrimaryKeyColumn(ctx, database, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to determine primary key: %v", err),
			})
		}

		// Build update query
		setStatements := []string{}
		values := []interface{}{}
		paramCounter := 1

		for column, value := range data {
			setStatements = append(setStatements, fmt.Sprintf("%s = $%d", 
				pgx.Identifier{column}.Sanitize(), paramCounter))
			values = append(values, value)
			paramCounter++
		}

		if len(setStatements) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No fields provided for update",
			})
		}

		// Add the primary key value for the WHERE clause
		values = append(values, idParam)

		// Create the UPDATE query
		query := fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s = $%d RETURNING *",
			pgx.Identifier{tableName}.Sanitize(),
			strings.Join(setStatements, ", "),
			pgx.Identifier{primaryKeyColumn}.Sanitize(),
			paramCounter,
		)

		// Execute the query
		row := database.QueryRow(ctx, query, values...)
		result, err := pgxRowToJSON(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Row with ID %s not found", idParam),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to update row: %v", err),
			})
		}

		return c.JSON(result)
	}
}

// DeleteTableRow deletes a row from the specified table
func DeleteTableRow(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tableName := c.Params("table")
		idParam := c.Params("id")
		ctx := context.Background()

		// Check RLS policies for the current user
		user := c.Locals("user")
		allowed, err := middleware.CheckRLS(database, user, tableName, "delete")
		if err != nil || !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access denied by row-level security policy",
			})
		}

		// Get primary key column
		primaryKeyColumn, err := getPrimaryKeyColumn(ctx, database, tableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to determine primary key: %v", err),
			})
		}

		// Create the DELETE query
		query := fmt.Sprintf(
			"DELETE FROM %s WHERE %s = $1 RETURNING *",
			pgx.Identifier{tableName}.Sanitize(),
			pgx.Identifier{primaryKeyColumn}.Sanitize(),
		)

		// Execute the query
		row := database.QueryRow(ctx, query, idParam)
		result, err := pgxRowToJSON(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Row with ID %s not found", idParam),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to delete row: %v", err),
			})
		}

		return c.JSON(result)
	}
}

// Helper to convert pgx.Rows to JSON array
func pgxRowsToJSON(rows pgx.Rows) ([]map[string]interface{}, error) {
	// Get the column info
	fields := rows.FieldDescriptions()
	
	result := []map[string]interface{}{}
	
	// Iterate through the rows
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		
		// Create a map for this row
		row := make(map[string]interface{})
		
		// Set key/value pairs for each column
		for i, field := range fields {
			row[string(field.Name)] = values[i]
		}
		
		result = append(result, row)
	}
	
	return result, nil
}

// Helper to convert a single pgx.Row to JSON
func pgxRowToJSON(row pgx.Row) (map[string]interface{}, error) {
	// Get the underlying pgx.Rows interface
	rows := row.(interface{ Rows() pgx.Rows }).Rows()
	
	// Check if there are any rows
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	
	// Get column info
	fields := rows.FieldDescriptions()
	
	// Get the values for this row
	values, err := rows.Values()
	if err != nil {
		return nil, err
	}
	
	// Create a map for this row
	result := make(map[string]interface{})
	
	// Set key/value pairs for each column
	for i, field := range fields {
		result[string(field.Name)] = values[i]
	}
	
	return result, nil
}

// QueryFilter holds information for filtering database queries
type QueryFilter struct {
	wheres []string
	params []interface{}
}

// buildQueryFilters extracts filter parameters from the request
func buildQueryFilters(c *fiber.Ctx) QueryFilter {
	filters := QueryFilter{
		wheres: []string{},
		params: []interface{}{},
	}
	
	paramCounter := 1
	
	// Process all query parameters
	c.Context().QueryArgs().VisitAll(func(key, val []byte) {
		k := string(key)
		v := string(val)
		
		// Skip pagination and sorting parameters
		if k == "page" || k == "page_size" || k == "order_by" || k == "order_dir" {
			return
		}
		
		// Handle operators in column names
		parts := strings.Split(k, ".")
		if len(parts) == 2 {
			column := parts[0]
			operator := parts[1]
			
			switch operator {
			case "eq":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s = $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "neq":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s != $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "gt":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s > $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "gte":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s >= $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "lt":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s < $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "lte":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s <= $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, v)
				paramCounter++
				
			case "like":
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s LIKE $%d", 
					pgx.Identifier{column}.Sanitize(), paramCounter))
				filters.params = append(filters.params, "%"+v+"%")
				paramCounter++
				
			case "in":
				// Parse comma-separated values
				values := strings.Split(v, ",")
				placeholders := make([]string, len(values))
				
				for i, val := range values {
					placeholders[i] = fmt.Sprintf("$%d", paramCounter)
					filters.params = append(filters.params, val)
					paramCounter++
				}
				
				filters.wheres = append(filters.wheres, fmt.Sprintf("%s IN (%s)", 
					pgx.Identifier{column}.Sanitize(), 
					strings.Join(placeholders, ", ")))
			}
		} else {
			// Simple equality filter
			filters.wheres = append(filters.wheres, fmt.Sprintf("%s = $%d", 
				pgx.Identifier{k}.Sanitize(), paramCounter))
			filters.params = append(filters.params, v)
			paramCounter++
		}
	})
	
	return filters
}

// getPrimaryKeyColumn determines the primary key column for a table
func getPrimaryKeyColumn(ctx context.Context, database *db.DB, tableName string) (string, error) {
	query := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass
		AND i.indisprimary;
	`
	
	var primaryKeyColumn string
	err := database.QueryRow(ctx, query, tableName).Scan(&primaryKeyColumn)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Fallback to id column if no primary key is defined
			return "id", nil
		}
		return "", err
	}
	
	return primaryKeyColumn, nil
}
