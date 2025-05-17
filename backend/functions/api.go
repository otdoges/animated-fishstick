package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackson/supabase-go/config"
	"github.com/jackson/supabase-go/db"
	"github.com/jackson/supabase-go/middleware"
	"github.com/jackson/supabase-go/routes"
)

var app *fiber.App
var database *db.DB

func init() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	// Initialize database
	database, err = db.Connect(cfg.Database)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}

	// Run migrations
	if err := database.RunMigrations(context.Background()); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
	}

	// Create Fiber app
	app = fiber.New(fiber.Config{
		AppName:               "Supabase Go",
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Authentication middleware
	app.Use(middleware.ClerkAuth(cfg.Auth))

	// Set up routes
	routes.Setup(app, database)
}

// Handler is the Lambda handler for Netlify Functions
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Convert API Gateway request to HTTP request
	httpRequest, err := apiGatewayRequestToHTTP(req)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error converting request: %v", err),
		}, nil
	}

	// Adjust path for Netlify
	path := httpRequest.URL.Path
	if strings.HasPrefix(path, "/.netlify/functions/api") {
		httpRequest.URL.Path = strings.TrimPrefix(path, "/.netlify/functions/api")
	}

	// Handle request using Fiber
	resp, err := app.Test(httpRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error handling request: %v", err),
		}, nil
	}

	// Convert HTTP response to API Gateway response
	return httpResponseToAPIGateway(resp)
}

// apiGatewayRequestToHTTP converts an API Gateway request to an HTTP request
func apiGatewayRequestToHTTP(req events.APIGatewayProxyRequest) (*http.Request, error) {
	// Create a new HTTP request
	httpReq, err := http.NewRequest(req.HTTPMethod, req.Path, strings.NewReader(req.Body))
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range req.Headers {
		httpReq.Header.Add(key, value)
	}

	// Add query parameters
	q := httpReq.URL.Query()
	for key, value := range req.QueryStringParameters {
		q.Add(key, value)
	}
	httpReq.URL.RawQuery = q.Encode()

	return httpReq, nil
}

// httpResponseToAPIGateway converts an HTTP response to an API Gateway response
func httpResponseToAPIGateway(resp *http.Response) (events.APIGatewayProxyResponse, error) {
	// Read the body
	defer resp.Body.Close()
	var body string
	if resp.ContentLength > 0 {
		buf := make([]byte, resp.ContentLength)
		_, err := resp.Body.Read(buf)
		if err != nil {
			return events.APIGatewayProxyResponse{}, err
		}
		body = string(buf)
	}

	// Convert headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Set CORS headers for all responses
	headers["Access-Control-Allow-Origin"] = "*"
	headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization"
	headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, PATCH, OPTIONS"

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}

func main() {
	lambda.Start(Handler)
}
