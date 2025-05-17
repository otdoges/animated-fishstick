package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackson/supabase-go/config"
)

// DB represents a PostgreSQL database connection pool
type DB struct {
	pool   *pgxpool.Pool
	config config.DatabaseConfig
}

// Connect establishes a connection to the PostgreSQL database
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	// Construct connection string
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// Set pool configuration values
	poolConfig.MaxConns = int32(cfg.Pool.MaxConn)
	poolConfig.MinConns = int32(cfg.Pool.MaxIdle)
	poolConfig.MaxConnLifetime = time.Duration(cfg.Pool.MaxLifetime) * time.Minute

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection is working
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool:   pool,
		config: cfg,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Exec executes a SQL query with the given arguments
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgx.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// Query executes a query that returns rows
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// Begin starts a new transaction
func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// GetTables returns a list of all tables in the database
func (db *DB) GetTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// GetTableColumns returns information about columns for a specific table
func (db *DB) GetTableColumns(ctx context.Context, tableName string) ([]Column, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var col Column
		var nullable, defaultVal, maxLength pgx.NullableText

		if err := rows.Scan(
			&col.Name,
			&col.DataType,
			&nullable,
			&defaultVal,
			&maxLength,
		); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.IsNullable = nullable.String == "YES"
		
		if defaultVal.Valid {
			col.Default = defaultVal.String
		}
		
		if maxLength.Valid {
			var length int
			fmt.Sscanf(maxLength.String, "%d", &length)
			col.MaxLength = length
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	return columns, nil
}

// Column represents database column information
type Column struct {
	Name       string
	DataType   string
	IsNullable bool
	Default    string
	MaxLength  int
}
