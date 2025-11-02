package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"eve.evalgo.org/registry"
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

	// Auto-register with registry service if REGISTRYSERVICE_API_URL is set
	portInt, _ := strconv.Atoi(port)
	if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
		ServiceID:    "infisicalservice",
		ServiceName:  "Infisical Secrets Management Service",
		Description:  "Secure secrets management using Infisical with semantic action support",
		Port:         portInt,
		Directory:    "/home/opunix/infisicalservice",
		Binary:       "infisicalservice",
		Capabilities: []string{"secrets-management", "infisical", "semantic-actions"},
	}); err != nil {
		log.Printf("Failed to register with registry: %v", err)
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Infisical Semantic Service on port %s", port)
		log.Println("Supports Infisical secrets management with Schema.org semantic types")
		log.Println("Environment variables:")
		log.Printf("  - INFISICAL_CLIENT_ID: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_ID")))
		log.Printf("  - INFISICAL_CLIENT_SECRET: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_SECRET")))

		if err := e.Start(":" + port); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Unregister from registry
	if err := registry.AutoUnregister("infisicalservice"); err != nil {
		log.Printf("Failed to unregister from registry: %v", err)
	}

	// Shutdown server
	if err := e.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
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
