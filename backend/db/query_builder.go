package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackson/supabase-go/pkg/querybuilder"
)

// QueryBuilder helps build and execute queries with filtering, sorting, and pagination
type QueryBuilder struct {
	db         *DB
	tableName  string
	selectCols []string
	joinClause string
}

// NewQueryBuilder creates a new QueryBuilder for the specified table
func (db *DB) NewQueryBuilder(tableName string) *QueryBuilder {
	return &QueryBuilder{
		db:         db,
		tableName:  tableName,
		selectCols: []string{"*"},
	}
}

// Select specifies the columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	if len(columns) > 0 {
		qb.selectCols = columns
	}
	return qb
}

// Join adds a JOIN clause to the query
func (qb *QueryBuilder) Join(joinClause string) *QueryBuilder {
	qb.joinClause = joinClause
	return qb
}

// Find executes the query and scans the results into dest
func (qb *QueryBuilder) Find(ctx context.Context, queryParams *querybuilder.QueryParams, dest interface{}) error {
	// Build the base query
	query := fmt.Sprintf("SELECT %s FROM \"%s\" t", strings.Join(qb.selectCols, ", "), qb.tableName)
	var args []interface{}

	// Add JOIN clause if specified
	if qb.joinClause != "" {
		query += " " + qb.joinClause
	}

	// Add WHERE clause
	whereClause, whereArgs := queryParams.BuildWhereClause()
	if whereClause != "" {
		query += " " + whereClause
		args = append(args, whereArgs...)
	}

	// Add ORDER BY clause
	orderByClause := queryParams.BuildOrderByClause()
	if orderByClause != "" {
		query += " " + orderByClause
	}

	// Add pagination
	paginationClause, paginationArgs := queryParams.BuildPaginationClause()
	if paginationClause != "" {
		query += " " + paginationClause
		args = append(args, paginationArgs...)
	}

	// Execute the query
	rows, err := qb.db.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Scan the results
	return pgxscan.ScanAll(dest, rows)
}

// Count executes a COUNT query with the same filters
func (qb *QueryBuilder) Count(ctx context.Context, queryParams *querybuilder.QueryParams) (int, error) {
	// Build the base query
	query := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\" t", qb.tableName)
	var args []interface{}

	// Add JOIN clause if specified
	if qb.joinClause != "" {
		query += " " + qb.joinClause
	}

	// Add WHERE clause
	whereClause, whereArgs := queryParams.BuildWhereClause()
	if whereClause != "" {
		query += " " + whereClause
		args = append(args, whereArgs...)
	}

	// Execute the query
	var count int
	err := qb.db.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting rows: %w", err)
	}

	return count, nil
}

// FindOne executes the query and scans the first result into dest
func (qb *QueryBuilder) FindOne(ctx context.Context, queryParams *querybuilder.QueryParams, dest interface{}) error {
	// Set limit to 1 for single result
	queryParams.Limit = 1

	// Build the base query
	query := fmt.Sprintf("SELECT %s FROM \"%s\" t", strings.Join(qb.selectCols, ", "), qb.tableName)
	var args []interface{}

	// Add JOIN clause if specified
	if qb.joinClause != "" {
		query += " " + qb.joinClause
	}

	// Add WHERE clause
	whereClause, whereArgs := queryParams.BuildWhereClause()
	if whereClause != "" {
		query += " " + whereClause
		args = append(args, whereArgs...)
	}

	// Add ORDER BY clause
	orderByClause := queryParams.BuildOrderByClause()
	if orderByClause != "" {
		query += " " + orderByClause
	}

	// Add LIMIT 1
	query += " LIMIT 1"

	// Execute the query
	rows, err := qb.db.pool.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Scan the result
	return pgxscan.ScanOne(dest, rows)
}
