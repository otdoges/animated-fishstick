# Query Builder

A flexible query builder for handling filtering, sorting, and pagination in Go applications using PostgreSQL.

## Features

- **Filtering**: Support for equality, comparison, and LIKE operators
- **Sorting**: Single or multiple column sorting with direction
- **Pagination**: Limit and offset support
- **Search**: Full-text search across all text columns
- **Security**: Protection against SQL injection

## Installation

```bash
go get github.com/your-org/your-repo/pkg/querybuilder
```

## Usage

### 1. Parse Query Parameters

```go
import (
    "net/url"
    "github.com/your-org/your-repo/pkg/querybuilder"
)

// Parse URL query parameters
values, _ := url.ParseQuery("name=John&age__gt=25&sort_by=created_at&sort_order=desc&limit=10&offset=0")
queryParams := querybuilder.ParseQueryParams(values)
```

### 2. Build SQL Clauses

```go
// Build WHERE clause
whereClause, whereArgs := queryParams.BuildWhereClause()

// Build ORDER BY clause
orderByClause := queryParams.BuildOrderByClause()

// Build pagination
paginationClause, paginationArgs := queryParams.BuildPaginationClause()
```

### 3. Use with Database Query

```go
// Example with pgx
query := fmt.Sprintf(`
    SELECT * FROM users t
    %s
    %s
    %s
`, whereClause, orderByClause, paginationClause)

// Combine all arguments
args := append(whereArgs, paginationArgs...)

// Execute query
rows, err := db.Query(ctx, query, args...)
```

## Query Parameters

### Filtering

- `field=value` - Equality filter
- `field__gt=value` - Greater than
- `field__gte=value` - Greater than or equal
- `field__lt=value` - Less than
- `field__lte=value` - Less than or equal
- `field__ne=value` - Not equal
- `field__like=value` - Case-insensitive LIKE with wildcards
- `field__in=value1,value2` - IN clause

### Sorting

- `sort_by=field` - Sort by field (default: `id`)
- `sort_order=asc|desc` - Sort direction (default: `asc`)

### Pagination

- `limit=10` - Number of records to return (default: 25)
- `offset=0` - Number of records to skip (default: 0)

### Search

- `q=search term` - Full-text search across all text columns

## Middleware

Use the provided middleware to automatically parse query parameters:

```go
import "github.com/your-org/your-repo/middleware"

app := fiber.New()
app.Use(middleware.QueryParamsMiddleware())

// In your handler
queryParams := c.Locals("queryParams").(*querybuilder.QueryParams)
```

## Query Builder

For a more fluent API, use the `QueryBuilder`:

```go
// Create a new query builder
qb := db.NewQueryBuilder("users").
    Select("id", "name", "email").
    Join("JOIN profiles p ON p.user_id = t.id")

// Find multiple records
var users []User
total, err := qb.Find(ctx, queryParams, &users)

// Find a single record
var user User
err := qb.FindOne(ctx, queryParams, &user)

// Count records
count, err := qb.Count(ctx, queryParams)
```

## License

MIT
