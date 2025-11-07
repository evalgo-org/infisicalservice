package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"eve.evalgo.org/common"
	evehttp "eve.evalgo.org/http"
	"eve.evalgo.org/registry"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize logger
	logger := common.ServiceLogger("infisicalservice", "1.0.0")

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// EVE health check
	e.GET("/health", evehttp.HealthCheckHandler("infisicalservice", "1.0.0"))

	// EVE API Key middleware
	apiKey := os.Getenv("INFISICAL_SERVICE_API_KEY")
	apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)
	e.POST("/v1/api/semantic/action", handleSemanticAction, apiKeyMiddleware)

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
		Capabilities: []string{"credential-management", "secrets-management", "infisical"},
	}); err != nil {
		logger.WithError(err).Error("Failed to register with registry")
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Starting Infisical Semantic Service on port %s", port)
		logger.Info("Supports Infisical secrets management with Schema.org semantic types")
		logger.Info("Environment variables:")
		logger.Infof("  - INFISICAL_CLIENT_ID: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_ID")))
		logger.Infof("  - INFISICAL_CLIENT_SECRET: %s", maskSecret(os.Getenv("INFISICAL_CLIENT_SECRET")))

		if err := e.Start(":" + port); err != nil {
			logger.WithError(err).Error("Server error")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Unregister from registry
	if err := registry.AutoUnregister("infisicalservice"); err != nil {
		logger.WithError(err).Error("Failed to unregister from registry")
	}

	// Shutdown server
	if err := e.Close(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}

	logger.Info("Server stopped")
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
