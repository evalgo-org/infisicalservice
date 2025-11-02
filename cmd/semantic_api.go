package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"eve.evalgo.org/semantic"
	"github.com/labstack/echo/v4"
)

// handleSemanticAction is the main handler for semantic action requests
func handleSemanticAction(c echo.Context) error {
	// Parse the JSON-LD request
	var rawAction map[string]interface{}
	if err := c.Bind(&rawAction); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON-LD request: " + err.Error(),
		})
	}

	// Determine action type
	actionType, ok := rawAction["@type"].(string)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "@type field is required",
		})
	}

	// Route to appropriate handler based on action type
	switch actionType {
	case "RetrieveAction":
		return handleRetrieveAction(c, rawAction)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Unsupported action type: %s. Supported types: RetrieveAction", actionType),
		})
	}
}

// handleRetrieveAction handles Infisical secret retrieval actions
func handleRetrieveAction(c echo.Context, rawAction map[string]interface{}) error {
	// Parse into InfisicalRetrieveAction
	actionBytes, _ := json.Marshal(rawAction)
	var action semantic.InfisicalRetrieveAction
	if err := json.Unmarshal(actionBytes, &action); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse RetrieveAction: " + err.Error(),
		})
	}

	// Validate required fields
	if action.Target == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target (InfisicalProject) is required",
		})
	}

	if action.Target.Identifier == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target.identifier (project_id) is required",
		})
	}

	if action.Target.Environment == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target.environment is required",
		})
	}

	if action.Target.Url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "target.url (Infisical instance URL) is required",
		})
	}

	// Get Infisical credentials from environment
	clientID := os.Getenv("INFISICAL_CLIENT_ID")
	clientSecret := os.Getenv("INFISICAL_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Infisical credentials not configured. Set INFISICAL_CLIENT_ID and INFISICAL_CLIENT_SECRET",
		})
	}

	// Set action metadata
	action.StartTime = time.Now().Format(time.RFC3339)
	action.ActionStatus = "ActiveActionStatus"

	// Execute secret retrieval
	log.Printf("Retrieving secrets from Infisical project=%s environment=%s",
		action.Target.Identifier, action.Target.Environment)

	err := action.RetrieveSecrets(clientID, clientSecret)

	action.EndTime = time.Now().Format(time.RFC3339)

	if err != nil {
		log.Printf("Failed to retrieve secrets: %v", err)
		return c.JSON(http.StatusInternalServerError, action)
	}

	// Mask secret values in logs
	log.Printf("Successfully retrieved %d secrets", len(action.Result))
	for _, secret := range action.Result {
		log.Printf("  - %s: %s", secret.Name, maskSecretValue(secret.Value))
	}

	return c.JSON(http.StatusOK, action)
}

// maskSecretValue masks a secret value for logging
func maskSecretValue(value string) string {
	if len(value) <= 8 {
		return "***"
	}
	return value[:2] + "..." + value[len(value)-2:]
}
