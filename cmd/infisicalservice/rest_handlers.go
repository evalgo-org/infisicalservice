package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// REST endpoint request types

type CreateSecretRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Environment string `json:"environment,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	SecretPath  string `json:"secretPath,omitempty"`
}

type UpdateSecretRequest struct {
	Value       string `json:"value"`
	Environment string `json:"environment,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	SecretPath  string `json:"secretPath,omitempty"`
}

// registerRESTEndpoints adds REST endpoints that convert to semantic actions
func registerRESTEndpoints(apiGroup *echo.Group, apiKeyMiddleware echo.MiddlewareFunc) {
	// POST /v1/api/secrets - Create secret
	apiGroup.POST("/secrets", createSecretREST, apiKeyMiddleware)

	// GET /v1/api/secrets/:key - Retrieve secret
	apiGroup.GET("/secrets/:key", getSecretREST, apiKeyMiddleware)

	// PUT /v1/api/secrets/:key - Update secret
	apiGroup.PUT("/secrets/:key", updateSecretREST, apiKeyMiddleware)

	// DELETE /v1/api/secrets/:key - Delete secret
	apiGroup.DELETE("/secrets/:key", deleteSecretREST, apiKeyMiddleware)
}

// createSecretREST handles REST POST /v1/api/secrets
func createSecretREST(c echo.Context) error {
	var req CreateSecretRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid request: %v", err)})
	}

	if req.Key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}
	if req.Value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "value is required"})
	}

	// Convert to JSON-LD CreateAction
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "CreateAction",
		"object": map[string]interface{}{
			"@type":      "PropertyValue",
			"identifier": req.Key,
			"value":      req.Value,
		},
	}

	// Add target with Infisical configuration
	target := map[string]interface{}{
		"@type": "EntryPoint",
	}
	if req.ProjectID != "" {
		target["actionPlatform"] = req.ProjectID
	}
	if req.Environment != "" {
		target["actionApplication"] = req.Environment
	}
	if req.SecretPath != "" {
		target["urlTemplate"] = req.SecretPath
	}
	if len(target) > 1 { // More than just @type
		action["target"] = target
	}

	return callSemanticHandler(c, action)
}

// getSecretREST handles REST GET /v1/api/secrets/:key
func getSecretREST(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}

	environment := c.QueryParam("environment")
	projectID := c.QueryParam("projectId")
	secretPath := c.QueryParam("secretPath")

	// Convert to JSON-LD SearchAction (which semantic handler interprets as RetrieveAction)
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "SearchAction",
		"query":    key,
	}

	// Add target with Infisical configuration
	target := map[string]interface{}{
		"@type": "EntryPoint",
	}
	if projectID != "" {
		target["actionPlatform"] = projectID
	}
	if environment != "" {
		target["actionApplication"] = environment
	}
	if secretPath != "" {
		target["urlTemplate"] = secretPath
	}
	if len(target) > 1 { // More than just @type
		action["target"] = target
	}

	return callSemanticHandler(c, action)
}

// updateSecretREST handles REST PUT /v1/api/secrets/:key
func updateSecretREST(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}

	var req UpdateSecretRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid request: %v", err)})
	}

	if req.Value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "value is required"})
	}

	// Convert to JSON-LD UpdateAction
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "UpdateAction",
		"object": map[string]interface{}{
			"@type":      "PropertyValue",
			"identifier": key,
			"value":      req.Value,
		},
	}

	// Add target with Infisical configuration
	target := map[string]interface{}{
		"@type": "EntryPoint",
	}
	if req.ProjectID != "" {
		target["actionPlatform"] = req.ProjectID
	}
	if req.Environment != "" {
		target["actionApplication"] = req.Environment
	}
	if req.SecretPath != "" {
		target["urlTemplate"] = req.SecretPath
	}
	if len(target) > 1 { // More than just @type
		action["target"] = target
	}

	return callSemanticHandler(c, action)
}

// deleteSecretREST handles REST DELETE /v1/api/secrets/:key
func deleteSecretREST(c echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key is required"})
	}

	environment := c.QueryParam("environment")
	projectID := c.QueryParam("projectId")
	secretPath := c.QueryParam("secretPath")

	// Convert to JSON-LD DeleteAction
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "DeleteAction",
		"object": map[string]interface{}{
			"@type":      "PropertyValue",
			"identifier": key,
		},
	}

	// Add target with Infisical configuration
	target := map[string]interface{}{
		"@type": "EntryPoint",
	}
	if projectID != "" {
		target["actionPlatform"] = projectID
	}
	if environment != "" {
		target["actionApplication"] = environment
	}
	if secretPath != "" {
		target["urlTemplate"] = secretPath
	}
	if len(target) > 1 { // More than just @type
		action["target"] = target
	}

	return callSemanticHandler(c, action)
}

// callSemanticHandler converts action to JSON and calls the semantic action handler
func callSemanticHandler(c echo.Context, action map[string]interface{}) error {
	// Marshal action to JSON
	actionJSON, err := json.Marshal(action)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to marshal action: %v", err)})
	}

	// Create new request with JSON-LD body
	newReq := c.Request().Clone(c.Request().Context())
	newReq.Body = io.NopCloser(bytes.NewReader(actionJSON))
	newReq.Header.Set("Content-Type", "application/json")

	// Create new context with modified request
	newCtx := c.Echo().NewContext(newReq, c.Response())
	newCtx.SetPath(c.Path())
	newCtx.SetParamNames(c.ParamNames()...)
	newCtx.SetParamValues(c.ParamValues()...)

	// Call the existing semantic action handler
	return handleSemanticAction(newCtx)
}
