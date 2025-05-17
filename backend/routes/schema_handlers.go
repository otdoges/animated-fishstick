package routes

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackson/supabase-go/db"
)

// SchemaInfo represents database schema information
type SchemaInfo struct {
	Tables []TableInfo `json:"tables"`
}

// TableInfo represents information about a database table
type TableInfo struct {
	Name    string        `json:"name"`
	Columns []db.Column   `json:"columns"`
	Indexes []IndexInfo   `json:"indexes"`
	FKeys   []ForeignKey  `json:"foreign_keys"`
}

// IndexInfo represents a database index
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name           string `json:"name"`
	ColumnName     string `json:"column_name"`
	RefTable       string `json:"reference_table"`
	RefColumnName  string `json:"reference_column"`
}

// GetDatabaseSchema returns the entire database schema
func GetDatabaseSchema(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		
		// Get all tables
		tables, err := database.GetTables(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get tables: %v", err),
			})
		}

		// Build schema information
		var schema SchemaInfo
		schema.Tables = make([]TableInfo, 0, len(tables))
		
		for _, tableName := range tables {
			tableInfo := TableInfo{
				Name: tableName,
			}
			
			// Get columns
			columns, err := database.GetTableColumns(ctx, tableName)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to get columns for table %s: %v", tableName, err),
				})
			}
			tableInfo.Columns = columns
			
			// Get indexes
			indexes, err := getTableIndexes(ctx, database, tableName)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to get indexes for table %s: %v", tableName, err),
				})
			}
			tableInfo.Indexes = indexes
			
			// Get foreign keys
			fkeys, err := getTableForeignKeys(ctx, database, tableName)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to get foreign keys for table %s: %v", tableName, err),
				})
			}
			tableInfo.FKeys = fkeys
			
			schema.Tables = append(schema.Tables, tableInfo)
		}
		
		return c.JSON(schema)
	}
}

// Helper function to get indexes for a table
func getTableIndexes(ctx context.Context, database *db.DB, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.relname AS index_name,
			array_agg(a.attname) AS column_names,
			ix.indisunique AS is_unique
		FROM
			pg_class t,
			pg_class i,
			pg_index ix,
			pg_attribute a
		WHERE
			t.oid = ix.indrelid
			AND i.oid = ix.indexrelid
			AND a.attrelid = t.oid
			AND a.attnum = ANY(ix.indkey)
			AND t.relkind = 'r'
			AND t.relname = $1
		GROUP BY
			i.relname,
			ix.indisunique;
	`
	
	rows, err := database.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var columnNamesArray []string
		
		if err := rows.Scan(&idx.Name, &columnNamesArray, &idx.Unique); err != nil {
			return nil, err
		}
		
		idx.Columns = columnNamesArray
		indexes = append(indexes, idx)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	return indexes, nil
}

// Helper function to get foreign keys for a table
func getTableForeignKeys(ctx context.Context, database *db.DB, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS reference_table,
			ccu.column_name AS reference_column
		FROM
			information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
		WHERE
			tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_name = $1;
	`
	
	rows, err := database.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var fkeys []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		
		if err := rows.Scan(&fk.Name, &fk.ColumnName, &fk.RefTable, &fk.RefColumnName); err != nil {
			return nil, err
		}
		
		fkeys = append(fkeys, fk)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	return fkeys, nil
}
