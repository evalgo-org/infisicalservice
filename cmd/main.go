package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"service": "infisicalservice",
			"status":  "healthy",
		})
	})

	// Check if API key is required
	apiKey := os.Getenv("INFISICAL_SERVICE_API_KEY")
	if apiKey != "" {
		log.Println("API key authentication enabled")
		// Apply API key middleware only to the semantic action endpoint
		apiKeyMiddleware := middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
			return key == apiKey, nil
		})
		e.POST("/v1/api/semantic/action", handleSemanticAction, apiKeyMiddleware)
	} else {
		log.Println("Running in development mode (no API key required)")
		e.POST("/v1/api/semantic/action", handleSemanticAction)
	}

	// Get port from environment or default to 8093
	port := os.Getenv("PORT")
	if port == "" {
		port = "8093"
	}

	log.Printf("Starting Infisical Semantic Service on port %s", port)
	log.Println("Supports Infisical secrets management with Schema.org semantic types")
	log.Println("Environment variables:")
	log.Printf("  - INFISICAL_CLIENT_ID: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_ID")))
	log.Printf("  - INFISICAL_CLIENT_SECRET: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_SECRET")))

	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}

// maskSecret masks a secret for logging
func maskSecret(secret string) string {
	if secret == "" {
		return "<not set>"
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
