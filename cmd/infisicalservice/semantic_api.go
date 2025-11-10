package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"eve.evalgo.org/semantic"
	infisical "github.com/infisical/go-sdk"
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
func handleRetrieveAction(c echo.Context, actionInterface interface{}) error {
	action, ok := actionInterface.(*semantic.SemanticAction)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid action type")
	}
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
	log.Printf("Retrieving secrets from Infisical (url=%s, project=%s, env=%s, path=%s, includeImports=%v)", url, projectID, environment, secretPath, includeImports)

	// Use EVE's Infisical integration to fetch secrets
	secrets, err := fetchSecretsFromInfisical(url, clientID, clientSecret, projectID, environment, secretPath, includeImports)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to retrieve secrets from Infisical", err)
	}

	log.Printf("Successfully retrieved %d secrets", len(secrets))

	// Store result using semantic Result structure
	// This follows Schema.org Dataset pattern with credentials as PropertyValues
	action.Result = &semantic.SemanticResult{
		Type:   "Dataset",
		Format: "application/json",
		Value:  secrets, // Structured data as array of {name, value} maps
		Schema: &semantic.ResultSchema{
			Type: "PropertyValueList",
			Properties: []semantic.PropertyValueSpec{
				{Type: "PropertyValue", Name: "name", ValueType: "Text", Description: "Secret key name"},
				{Type: "PropertyValue", Name: "value", ValueType: "Text", Description: "Secret value"},
			},
		},
	}

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

// fetchSecretsFromInfisical retrieves secrets from Infisical using the Go SDK
func fetchSecretsFromInfisical(url, clientID, clientSecret, projectID, environment, secretPath string, includeImports bool) ([]interface{}, error) {
	// Extract host from URL (remove https:// prefix)
	host := strings.TrimPrefix(url, "https://")
	host = strings.TrimPrefix(host, "http://")

	// Create Infisical client
	client := infisical.NewInfisicalClient(context.Background(), infisical.Config{
		SiteUrl:          "https://" + host,
		AutoTokenRefresh: false,
	})

	// Authenticate
	_, err := client.Auth().UniversalAuthLogin(clientID, clientSecret)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	log.Printf("DEBUG: Fetching secrets with SDK - projectID=%s, env=%s, path=%s, includeImports=%v", projectID, environment, secretPath, includeImports)

	// Fetch secrets
	apiKeySecrets, err := client.Secrets().List(infisical.ListSecretsOptions{
		AttachToProcessEnv: false,
		Environment:        environment,
		ProjectID:          projectID,
		SecretPath:         secretPath,
		IncludeImports:     includeImports,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	log.Printf("DEBUG: SDK returned %d secrets", len(apiKeySecrets))

	// Convert to semantic.PropertyValue format
	secrets := make([]interface{}, len(apiKeySecrets))
	for idx, secret := range apiKeySecrets {
		secrets[idx] = map[string]string{
			"name":  secret.SecretKey,
			"value": secret.SecretValue,
		}
		log.Printf("Retrieved secret: %s = %s", secret.SecretKey, maskSecretValue(secret.SecretValue))
	}

	return secrets, nil
}
