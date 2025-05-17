package middleware

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/jackson/supabase-go/pkg/querybuilder"
)

// QueryParamsMiddleware parses and validates query parameters
func QueryParamsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse query parameters
		values, err := url.ParseQuery(string(c.Request().URI().QueryString()))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid query parameters",
			})
		}

		// Parse into QueryParams struct
		queryParams := querybuilder.ParseQueryParams(values)

		// Store in context for handlers to use
		c.Locals("queryParams", queryParams)

		// Continue to next handler
		return c.Next()
	}
}
