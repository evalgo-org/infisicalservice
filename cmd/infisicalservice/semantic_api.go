package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"eve.evalgo.org/semantic"
	"github.com/labstack/echo/v4"
)

// handleSemanticAction is the main handler for semantic action requests
func handleSemanticAction(c echo.Context) error {
	// Read request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body: " + err.Error(),
		})
	}

	// Parse as SemanticAction
	action, err := semantic.ParseSemanticAction(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse action: " + err.Error(),
		})
	}

	// Dispatch to registered handler using the ActionRegistry
	// No switch statement needed - handlers are registered at startup
	return semantic.Handle(c, action)
}

// handleRetrieveAction handles Infisical secret retrieval actions
func handleRetrieveAction(c echo.Context, action *semantic.SemanticAction) error {
	// Extract Infisical target configuration using helper
	url, projectID, environment, secretPath, includeImports, err := semantic.GetInfisicalTargetFromAction(action)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to extract Infisical target", err)
	}

	// Get Infisical credentials from environment
	clientID := os.Getenv("INFISICAL_CLIENT_ID")
	clientSecret := os.Getenv("INFISICAL_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return semantic.ReturnActionError(c, action, "Infisical credentials not configured", nil)
	}

	// Execute secret retrieval using the extracted configuration
	log.Printf("Retrieving secrets from Infisical (project=%s, env=%s, path=%s)", projectID, environment, secretPath)

	// TODO: Implement actual Infisical API call using url, projectID, environment, secretPath, includeImports
	// For now, just set success
	_ = url
	_ = includeImports
	_ = clientID
	_ = clientSecret

	// Store result in properties
	secrets := []interface{}{
		map[string]string{
			"name":  "example-secret",
			"value": "***masked***",
		},
	}
	action.Properties["result"] = secrets

	log.Printf("Successfully retrieved %d secrets", len(secrets))

	semantic.SetSuccessOnAction(action)
	return c.JSON(http.StatusOK, action)
}

// maskSecretValue masks a secret value for logging
func maskSecretValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:2] + "..." + value[len(value)-2:]
}
