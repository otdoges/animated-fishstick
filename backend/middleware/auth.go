package middleware

import (
	"errors"
	"fmt"
	"strings"

	"github.com/clerkinc/clerk-sdk-go"
	"github.com/gofiber/fiber/v2"
	"github.com/jackson/supabase-go/config"
)

// ClerkAuth returns a middleware that handles authentication using Clerk
func ClerkAuth(cfg config.AuthConfig) fiber.Handler {
	// Initialize Clerk client
	client, err := clerk.NewClient(cfg.ClerkSecretKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Clerk client: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// Skip authentication for specific routes if needed
		path := c.Path()
		if isPublicRoute(path) {
			return c.Next()
		}

		// Get authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Parse token from header
		token := parseAuthHeader(authHeader)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		// Verify the JWT token
		claims, err := client.VerifyToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Get session and user information
		sessions, err := client.Sessions().GetAll(&clerk.GetSessionsParams{
			ClientToken: token,
		})
		if err != nil || len(sessions) == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session not found",
			})
		}

		// Get the user
		user, err := client.Users().Read(claims.GetSubject())
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Set user and session info in context for use in route handlers
		c.Locals("user", user)
		c.Locals("session", sessions[0])
		c.Locals("userId", user.ID)
		c.Locals("userRole", getUserRole(user))

		return c.Next()
	}
}

// parseAuthHeader extracts the token from the authorization header
func parseAuthHeader(authHeader string) string {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// isPublicRoute checks if a route should be publicly accessible
func isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/api/health",
		"/api/status",
		"/api/schema",
	}

	for _, route := range publicRoutes {
		if path == route {
			return true
		}
	}

	return false
}

// getUserRole extracts user role from Clerk user metadata
func getUserRole(user *clerk.User) string {
	// Default role
	defaultRole := "user"

	// Check if public metadata has a role
	if user.PublicMetadata != nil {
		if role, ok := user.PublicMetadata["role"].(string); ok {
			return role
		}
	}

	return defaultRole
}

// CheckRLS checks if the current user has access to the requested resource
// based on Row Level Security policies
func CheckRLS(db interface{}, user interface{}, resource string, action string) (bool, error) {
	// This is a placeholder for actual RLS check implementation
	// In a real implementation, this would query the database for RLS policies
	// and evaluate them against the current user and requested resource
	
	if user == nil {
		return false, errors.New("user not authenticated")
	}

	// For now, we'll just allow all authenticated users
	return true, nil
}
