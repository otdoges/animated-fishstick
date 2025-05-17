package routes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackson/supabase-go/db"
)

// RLSPolicy represents a row-level security policy
type RLSPolicy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TableName   string    `json:"table_name"`
	Action      string    `json:"action"`
	Roles       []string  `json:"roles"`
	Definition  string    `json:"definition"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Description string    `json:"description"`
}

// RLSPolicyRequest represents a request to create or update a RLS policy
type RLSPolicyRequest struct {
	Name        string   `json:"name"`
	TableName   string   `json:"table_name"`
	Action      string   `json:"action"`
	Roles       []string `json:"roles"`
	Definition  string   `json:"definition"`
	Description string   `json:"description"`
}

// GetRLSPolicies returns all RLS policies
func GetRLSPolicies(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		
		// Get table filter if any
		tableName := c.Query("table")
		
		query := `
			SELECT 
				id, name, table_name, action, roles, definition, 
				created_at, updated_at, description
			FROM rls_policies
		`
		
		var params []interface{}
		var paramCount int
		
		if tableName != "" {
			query += " WHERE table_name = $1"
			params = append(params, tableName)
			paramCount++
		}
		
		query += " ORDER BY table_name, name"
		
		rows, err := database.Query(ctx, query, params...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to query RLS policies: %v", err),
			})
		}
		defer rows.Close()
		
		var policies []RLSPolicy
		for rows.Next() {
			var policy RLSPolicy
			var rolesJson string
			
			err := rows.Scan(
				&policy.ID,
				&policy.Name,
				&policy.TableName,
				&policy.Action,
				&rolesJson,
				&policy.Definition,
				&policy.CreatedAt,
				&policy.UpdatedAt,
				&policy.Description,
			)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to scan policy: %v", err),
				})
			}
			
			// Parse roles from JSON
			policy.Roles = parseRoles(rolesJson)
			
			policies = append(policies, policy)
		}
		
		return c.JSON(fiber.Map{
			"policies": policies,
		})
	}
}

// GetRLSPolicy returns a specific RLS policy
func GetRLSPolicy(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		policyID := c.Params("id")
		
		query := `
			SELECT 
				id, name, table_name, action, roles, definition, 
				created_at, updated_at, description
			FROM rls_policies
			WHERE id = $1
		`
		
		row := database.QueryRow(ctx, query, policyID)
		
		var policy RLSPolicy
		var rolesJson string
		
		err := row.Scan(
			&policy.ID,
			&policy.Name,
			&policy.TableName,
			&policy.Action,
			&rolesJson,
			&policy.Definition,
			&policy.CreatedAt,
			&policy.UpdatedAt,
			&policy.Description,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Policy with ID %s not found", policyID),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get policy: %v", err),
			})
		}
		
		// Parse roles from JSON
		policy.Roles = parseRoles(rolesJson)
		
		return c.JSON(policy)
	}
}

// CreateRLSPolicy creates a new RLS policy
func CreateRLSPolicy(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		
		var req RLSPolicyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
		}
		
		// Validate request
		if req.Name == "" || req.TableName == "" || req.Action == "" || len(req.Roles) == 0 || req.Definition == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Name, table_name, action, roles, and definition are required",
			})
		}
		
		// Check if table exists
		tableExists, err := tableExists(ctx, database, req.TableName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to check table existence: %v", err),
			})
		}
		if !tableExists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Table '%s' does not exist", req.TableName),
			})
		}
		
		// Check if policy name is unique for the table
		policyExists, err := policyExists(ctx, database, req.Name, req.TableName, "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to check policy existence: %v", err),
			})
		}
		if policyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": fmt.Sprintf("Policy with name '%s' already exists for table '%s'", req.Name, req.TableName),
			})
		}
		
		// Create policy in the database
		query := `
			INSERT INTO rls_policies (
				name, table_name, action, roles, definition, description
			) VALUES (
				$1, $2, $3, $4, $5, $6
			) RETURNING 
				id, name, table_name, action, roles, definition, 
				created_at, updated_at, description
		`
		
		row := database.QueryRow(
			ctx,
			query,
			req.Name,
			req.TableName,
			req.Action,
			rolesArrayToJson(req.Roles),
			req.Definition,
			req.Description,
		)
		
		var policy RLSPolicy
		var rolesJson string
		
		err = row.Scan(
			&policy.ID,
			&policy.Name,
			&policy.TableName,
			&policy.Action,
			&rolesJson,
			&policy.Definition,
			&policy.CreatedAt,
			&policy.UpdatedAt,
			&policy.Description,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create policy: %v", err),
			})
		}
		
		// Parse roles from JSON
		policy.Roles = parseRoles(rolesJson)
		
		// Apply the policy to the table
		err = applyRLSPolicy(ctx, database, policy)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to apply policy: %v", err),
			})
		}
		
		return c.Status(fiber.StatusCreated).JSON(policy)
	}
}

// UpdateRLSPolicy updates an existing RLS policy
func UpdateRLSPolicy(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		policyID := c.Params("id")
		
		var req RLSPolicyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
		}
		
		// Check if policy exists
		var oldPolicy RLSPolicy
		err := getPolicy(ctx, database, policyID, &oldPolicy)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Policy with ID %s not found", policyID),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get policy: %v", err),
			})
		}
		
		// Check if name is unique (if changed)
		if req.Name != oldPolicy.Name || req.TableName != oldPolicy.TableName {
			policyExists, err := policyExists(ctx, database, req.Name, req.TableName, policyID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Failed to check policy existence: %v", err),
				})
			}
			if policyExists {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": fmt.Sprintf("Policy with name '%s' already exists for table '%s'", req.Name, req.TableName),
				})
			}
		}
		
		// Prepare update query
		query := `
			UPDATE rls_policies
			SET 
				name = $1,
				table_name = $2,
				action = $3,
				roles = $4,
				definition = $5,
				description = $6,
				updated_at = NOW()
			WHERE id = $7
			RETURNING 
				id, name, table_name, action, roles, definition, 
				created_at, updated_at, description
		`
		
		row := database.QueryRow(
			ctx,
			query,
			req.Name,
			req.TableName,
			req.Action,
			rolesArrayToJson(req.Roles),
			req.Definition,
			req.Description,
			policyID,
		)
		
		var policy RLSPolicy
		var rolesJson string
		
		err = row.Scan(
			&policy.ID,
			&policy.Name,
			&policy.TableName,
			&policy.Action,
			&rolesJson,
			&policy.Definition,
			&policy.CreatedAt,
			&policy.UpdatedAt,
			&policy.Description,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to update policy: %v", err),
			})
		}
		
		// Parse roles from JSON
		policy.Roles = parseRoles(rolesJson)
		
		// Drop the old policy
		err = dropRLSPolicy(ctx, database, oldPolicy)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to drop old policy: %v", err),
			})
		}
		
		// Apply the updated policy
		err = applyRLSPolicy(ctx, database, policy)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to apply updated policy: %v", err),
			})
		}
		
		return c.JSON(policy)
	}
}

// DeleteRLSPolicy deletes a RLS policy
func DeleteRLSPolicy(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.Background()
		policyID := c.Params("id")
		
		// Get the policy first to drop it from PostgreSQL
		var policy RLSPolicy
		err := getPolicy(ctx, database, policyID, &policy)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": fmt.Sprintf("Policy with ID %s not found", policyID),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get policy: %v", err),
			})
		}
		
		// Drop the policy
		err = dropRLSPolicy(ctx, database, policy)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to drop policy: %v", err),
			})
		}
		
		// Delete from our policies table
		query := "DELETE FROM rls_policies WHERE id = $1"
		_, err = database.Exec(ctx, query, policyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to delete policy: %v", err),
			})
		}
		
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// Helper function to check if a table exists
func tableExists(ctx context.Context, database *db.DB, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`
	
	var exists bool
	err := database.QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

// Helper function to check if a policy exists
func policyExists(ctx context.Context, database *db.DB, name, tableName, excludeID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM rls_policies 
			WHERE name = $1 
			AND table_name = $2
	`
	
	var params []interface{}
	params = append(params, name, tableName)
	
	if excludeID != "" {
		query += " AND id != $3"
		params = append(params, excludeID)
	}
	
	query += ")"
	
	var exists bool
	err := database.QueryRow(ctx, query, params...).Scan(&exists)
	return exists, err
}

// Helper function to get a policy by ID
func getPolicy(ctx context.Context, database *db.DB, policyID string, policy *RLSPolicy) error {
	query := `
		SELECT 
			id, name, table_name, action, roles, definition, 
			created_at, updated_at, description
		FROM rls_policies
		WHERE id = $1
	`
	
	row := database.QueryRow(ctx, query, policyID)
	
	var rolesJson string
	err := row.Scan(
		&policy.ID,
		&policy.Name,
		&policy.TableName,
		&policy.Action,
		&rolesJson,
		&policy.Definition,
		&policy.CreatedAt,
		&policy.UpdatedAt,
		&policy.Description,
	)
	
	if err != nil {
		return err
	}
	
	// Parse roles from JSON
	policy.Roles = parseRoles(rolesJson)
	return nil
}

// Helper function to apply a RLS policy to the database
func applyRLSPolicy(ctx context.Context, database *db.DB, policy RLSPolicy) error {
	// First enable row-level security on the table
	enableQuery := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY",
		pgx.Identifier{policy.TableName}.Sanitize())
	
	_, err := database.Exec(ctx, enableQuery)
	if err != nil {
		return fmt.Errorf("failed to enable RLS on table: %w", err)
	}
	
	// Create the policy
	createQuery := fmt.Sprintf(
		"CREATE POLICY %s ON %s FOR %s TO %s USING (%s)",
		pgx.Identifier{policy.Name}.Sanitize(),
		pgx.Identifier{policy.TableName}.Sanitize(),
		strings.ToUpper(policy.Action),
		strings.Join(policy.Roles, ", "),
		policy.Definition,
	)
	
	_, err = database.Exec(ctx, createQuery)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	
	return nil
}

// Helper function to drop a RLS policy
func dropRLSPolicy(ctx context.Context, database *db.DB, policy RLSPolicy) error {
	dropQuery := fmt.Sprintf(
		"DROP POLICY IF EXISTS %s ON %s",
		pgx.Identifier{policy.Name}.Sanitize(),
		pgx.Identifier{policy.TableName}.Sanitize(),
	)
	
	_, err := database.Exec(ctx, dropQuery)
	return err
}

// Helper function to convert roles array to JSON string
func rolesArrayToJson(roles []string) string {
	// Simple JSON array representation
	return "[\"" + strings.Join(roles, "\",\"") + "\"]"
}

// Helper function to parse roles from JSON string
func parseRoles(rolesJson string) []string {
	// Remove brackets and quotes
	trimmed := strings.Trim(rolesJson, "[]")
	if trimmed == "" {
		return []string{}
	}
	
	// Split by delimiter and clean up
	parts := strings.Split(trimmed, ",")
	var roles []string
	
	for _, part := range parts {
		role := strings.Trim(part, "\" ")
		if role != "" {
			roles = append(roles, role)
		}
	}
	
	return roles
}
