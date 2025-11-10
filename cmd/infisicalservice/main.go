package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"eve.evalgo.org/web"

	"eve.evalgo.org/common"
	evehttp "eve.evalgo.org/http"
	"eve.evalgo.org/registry"
	"eve.evalgo.org/semantic"
	"eve.evalgo.org/statemanager"
	"eve.evalgo.org/tracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize logger
	logger := common.ServiceLogger("infisicalservice", "1.0.0")

	// Register action handlers with the semantic action registry
	// This allows the service to handle semantic actions without modifying switch statements
	semantic.MustRegister("RetrieveAction", handleRetrieveAction)

	// Create Echo instance
	e := echo.New()

	// Register EVE corporate identity assets
	web.RegisterAssets(e)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize tracing (gracefully disabled if unavailable)
	if tracer := tracing.Init(tracing.InitConfig{
		ServiceID:        "infisicalservice",
		DisableIfMissing: true,
	}); tracer != nil {
		e.Use(tracer.Middleware())
	}

	// EVE health check
	e.GET("/health", evehttp.HealthCheckHandler("infisicalservice", "1.0.0"))

	// Documentation endpoint
	e.GET("/v1/api/docs", evehttp.DocumentationHandler(evehttp.ServiceDocConfig{
		ServiceID:           "infisicalservice",
		ServiceName:         "Infisical Secrets Management Service",
		Description:         "Secure secrets management using Infisical with semantic action support",
		Version:             "v1",
		Port:                8093,
		IncludeDependencies: true,
		Capabilities:        []string{"credential-management", "secrets-management", "infisical", "state-tracking"},
		Endpoints: []evehttp.EndpointDoc{
			{
				Method:      "POST",
				Path:        "/v1/api/semantic/action",
				Description: "Execute secrets management operations via semantic actions (primary interface)",
			},
			{
				Method:      "POST",
				Path:        "/v1/api/secrets",
				Description: "Create secret (REST convenience - converts to CreateAction)",
			},
			{
				Method:      "GET",
				Path:        "/v1/api/secrets/:key",
				Description: "Retrieve secret (REST convenience - converts to SearchAction)",
			},
			{
				Method:      "PUT",
				Path:        "/v1/api/secrets/:key",
				Description: "Update secret (REST convenience - converts to UpdateAction)",
			},
			{
				Method:      "DELETE",
				Path:        "/v1/api/secrets/:key",
				Description: "Delete secret (REST convenience - converts to DeleteAction)",
			},
			{
				Method:      "GET",
				Path:        "/health",
				Description: "Health check endpoint",
			},
		},
	}))

	// Initialize state manager
	sm := statemanager.New(statemanager.Config{
		ServiceName:   "infisicalservice",
		MaxOperations: 100,
	})

	// Register state endpoints
	apiGroup := e.Group("/v1/api")
	sm.RegisterRoutes(apiGroup)

	// EVE API Key middleware
	apiKey := os.Getenv("INFISICAL_SERVICE_API_KEY")
	apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)

	// Semantic action endpoint (primary interface)
	apiGroup.POST("/semantic/action", handleSemanticAction, apiKeyMiddleware)

	// REST endpoints (convenience adapters that convert to semantic actions)
	registerRESTEndpoints(apiGroup, apiKeyMiddleware)

	// Get port from environment or default to 8093
	port := os.Getenv("PORT")
	if port == "" {
		port = "8093"
	}

	// Get service URL from environment (for Docker container names) or default to localhost
	portInt, _ := strconv.Atoi(port)
	serviceURL := os.Getenv("INFISICAL_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("http://localhost:%d", portInt)
	}

	// Auto-register with registry service if REGISTRYSERVICE_API_URL is set
	if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
		ServiceID:    "infisicalservice",
		ServiceName:  "Infisical Secrets Management Service",
		Description:  "Secure secrets management using Infisical with semantic action support",
		Port:         portInt,
		ServiceURL:   serviceURL,
		Directory:    "/home/opunix/infisicalservice",
		Binary:       "infisicalservice",
		Version:      "v1",
		Capabilities: []string{"credential-management", "secrets-management", "infisical", "state-tracking"},
		ActionCapabilities: []registry.ActionCapability{
			{
				ActionType:  "RetrieveAction",
				Description: "Retrieves credentials from Infisical secrets manager",
				ResultSchema: &semantic.ResultSchema{
					Type: "PropertyValueList",
					Properties: []semantic.PropertyValueSpec{
						{
							Type:        "PropertyValue",
							Name:        "name",
							ValueType:   "Text",
							Description: "Secret key name",
						},
						{
							Type:        "PropertyValue",
							Name:        "value",
							ValueType:   "Text",
							Description: "Secret value",
						},
					},
				},
			},
		},
		APIVersions: []registry.APIVersion{
			{
				Version:       "v1",
				URL:           fmt.Sprintf("%s/v1", serviceURL),
				Documentation: fmt.Sprintf("%s/v1/api/docs", serviceURL),
				IsDefault:     true,
				Status:        "stable",
				ReleaseDate:   "2024-01-01",
				Capabilities:  []string{"credential-management", "secrets-management", "infisical"},
			},
		},
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
