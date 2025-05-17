package querybuilder

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// QueryParams represents the query parameters for filtering, sorting, and pagination
type QueryParams struct {
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string // "asc" or "desc"
	Filters    map[string]string
	Search     string
}

// ParseQueryParams parses URL query parameters into a QueryParams struct
func ParseQueryParams(values url.Values) *QueryParams {
	qp := &QueryParams{
		Limit:     25, // Default limit
		Offset:    0,  // Default offset
		SortBy:    "id",
		SortOrder: "asc",
		Filters:   make(map[string]string),
	}

	// Parse limit
	if limitStr := values.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			qp.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := values.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			qp.Offset = offset
		}
	}

	// Parse sort
	sortBy := values.Get("sort_by")
	sortOrder := values.Get("sort_order")

	if sortBy != "" {
		qp.SortBy = sortBy
	}

	if sortOrder == "desc" || sortOrder == "DESC" {
		qp.SortOrder = "DESC"
	} else {
		qp.SortOrder = "ASC"
	}

	// Parse filters
	for key, values := range values {
		if len(values) > 0 && !isReservedParam(key) {
			qp.Filters[key] = values[0]
		}
	}

	// Parse search
	if search := values.Get("q"); search != "" {
		qp.Search = search
	}

	return qp
}

// BuildWhereClause builds a WHERE clause from the filters
func (qp *QueryParams) BuildWhereClause() (string, []interface{}) {
	if len(qp.Filters) == 0 && qp.Search == "" {
		return "", nil
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Add search condition if present
	if qp.Search != "" {
		// This is a simple search that looks in all text/varchar columns
		// You might want to customize this based on your needs
		conditions = append(conditions, "(to_tsvector('english', to_jsonb(t)::text) @@ plainto_tsquery('english', $1))")
		args = append(args, qp.Search)
		argIdx++
	}

	// Add filter conditions
	for field, value := range qp.Filters {
		// Skip empty values
		if value == "" {
			continue
		}

		// Handle special operators (e.g., field__gt, field__lt, etc.)
		if parts := strings.Split(field, "__"); len(parts) == 2 {
			fieldName := fmt.Sprintf("t.\"%s\"", parts[0])
			switch parts[1] {
			case "gt":
				conditions = append(conditions, fmt.Sprintf("%s > $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			case "gte":
				conditions = append(conditions, fmt.Sprintf("%s >= $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			case "lt":
				conditions = append(conditions, fmt.Sprintf("%s < $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			case "lte":
				conditions = append(conditions, fmt.Sprintf("%s <= $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			case "ne":
				conditions = append(conditions, fmt.Sprintf("%s != $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			case "like":
				conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", fieldName, argIdx))
				args = append(args, "%"+value+"%")
				argIdx++
			case "in":
				values := strings.Split(value, ",")
				placeholders := make([]string, len(values))
				for i, v := range values {
					placeholders[i] = fmt.Sprintf("$%d", argIdx)
					args = append(args, strings.TrimSpace(v))
					argIdx++
				}
				conditions = append(conditions, fmt.Sprintf("%s IN (%s)", fieldName, strings.Join(placeholders, ",")))
			default:
				// Default to equality
				conditions = append(conditions, fmt.Sprintf("%s = $%d", fieldName, argIdx))
				args = append(args, value)
				argIdx++
			}
		} else {
			// Default to equality
			conditions = append(conditions, fmt.Sprintf("t.\"%s\" = $%d", field, argIdx))
			args = append(args, value)
			argIdx++
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// BuildOrderByClause builds an ORDER BY clause
func (qp *QueryParams) BuildOrderByClause() string {
	if qp.SortBy == "" {
		return ""
	}

	// Prevent SQL injection by only allowing alphanumeric, underscore, and dot
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			return r
		}
		return -1
	}, qp.SortBy)

	// If the string is empty after sanitization, use a safe default
	if sanitized == "" {
		sanitized = "id"
	}

	orderBy := fmt.Sprintf("ORDER BY %s %s", sanitized, qp.SortOrder)

	// Always include a secondary sort on ID for consistent pagination
	if sanitized != "id" {
		orderBy += fmt.Sprintf(", id %s", qp.SortOrder)
	}

	return orderBy
}

// BuildPaginationClause builds LIMIT and OFFSET clauses
func (qp *QueryParams) BuildPaginationClause() (string, []interface{}) {
	if qp.Limit <= 0 && qp.Offset <= 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}

	if qp.Limit > 0 {
		clauses = append(clauses, fmt.Sprintf("LIMIT $%d", len(args)+1))
		args = append(args, qp.Limit)
	}

	if qp.Offset > 0 {
		clauses = append(clauses, fmt.Sprintf("OFFSET $%d", len(args)+1))
		args = append(args, qp.Offset)
	}

	return strings.Join(clauses, " "), args
}

// isReservedParam checks if a query parameter is reserved for pagination, sorting, etc.
func isReservedParam(key string) bool {
	reserved := []string{"limit", "offset", "sort_by", "sort_order", "q"}
	for _, r := range reserved {
		if strings.EqualFold(key, r) {
			return true
		}
	}
	return false
}
