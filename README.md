# Infisicalservice - Semantic Secrets Management Service

Infisicalservice is a Schema.org semantic web service for retrieving secrets from Infisical using JSON-LD workflows.

## Features

- **Schema.org Semantic Types**: Uses RetrieveAction and Project types
- **EVE v0.0.19 Integration**: Leverages EVE's Infisical semantic types
- **Multi-Project Support**: Retrieve secrets from different Infisical projects
- **Environment Isolation**: Separate secrets for dev/prod/staging
- **Secure Logging**: Automatic secret masking in logs
- **API Key Protection**: Optional API key authentication

## Architecture

```
when/fetcher → infisicalservice:8093 → Infisical API → Secrets
                       ↓
              EVE v0.0.19 semantic types
                       ↓
         Return secrets as PropertyValue array
```

## Environment Variables

### Required (stored on when scheduler host):
- `INFISICAL_CLIENT_ID`: Infisical Universal Auth client ID
- `INFISICAL_CLIENT_SECRET`: Infisical Universal Auth client secret

### Optional:
- `PORT`: Service port (default: 8093)
- `INFISICAL_SERVICE_API_KEY`: Enable API key authentication

## API

### Health Check
```bash
GET /health
```

Response:
```json
{
  "service": "infisicalservice",
  "status": "healthy"
}
```

### Retrieve Secrets
```bash
POST /v1/api/semantic/action
Content-Type: application/json
```

Request:
```json
{
  "@context": "https://schema.org",
  "@type": "RetrieveAction",
  "identifier": "get-s3-secrets",
  "name": "Retrieve S3 Credentials",
  "target": {
    "@type": "Project",
    "identifier": "your-project-id",
    "environment": "prod",
    "url": "https://app.infisical.com",
    "secretPath": "/",
    "includeImports": true
  }
}
```

Response:
```json
{
  "@context": "https://schema.org",
  "@type": "RetrieveAction",
  "identifier": "get-s3-secrets",
  "name": "Retrieve S3 Credentials",
  "target": {
    "@type": "Project",
    "identifier": "your-project-id",
    "environment": "prod",
    "url": "https://app.infisical.com"
  },
  "result": [
    {
      "@type": "PropertyValue",
      "name": "HETZNER_S3_ACCESS_KEY",
      "value": "actual-key-value"
    },
    {
      "@type": "PropertyValue",
      "name": "HETZNER_S3_SECRET_KEY",
      "value": "actual-secret-value"
    }
  ],
  "actionStatus": "CompletedActionStatus",
  "startTime": "2025-11-02T12:15:00+01:00",
  "endTime": "2025-11-02T12:15:01+01:00"
}
```

## Multi-Project Organization

Organize secrets by service using separate Infisical projects:

### iqs-s3-secrets
- **Environment**: prod, dev
- **Secrets**:
  - `HETZNER_S3_ACCESS_KEY`
  - `HETZNER_S3_SECRET_KEY`
  - `HETZNER_S3_URL`

### iqs-poolparty-secrets
- **Environment**: prod, dev
- **Secrets**:
  - `SPARQL_URL`
  - `SPARQL_USERNAME`
  - `SPARQL_PASSWORD`

### iqs-basex-secrets
- **Environment**: prod, dev
- **Secrets**:
  - `BASEX_URL`
  - `BASEX_USERNAME`
  - `BASEX_PASSWORD`

## Workflow Integration

Secrets are retrieved at the beginning of workflows and injected into subsequent actions:

```json
{
  "@type": "ItemList",
  "itemListElement": [
    {
      "position": 1,
      "item": {
        "@type": "RetrieveAction",
        "identifier": "get-secrets",
        "target": {
          "@type": "Project",
          "identifier": "iqs-s3-secrets",
          "environment": "prod"
        }
      }
    },
    {
      "position": 2,
      "item": {
        "@type": "CreateAction",
        "identifier": "upload-to-s3",
        "dependsOn": "get-secrets",
        "target": {
          "accessKey": "${secrets.HETZNER_S3_ACCESS_KEY}",
          "secretKey": "${secrets.HETZNER_S3_SECRET_KEY}"
        }
      }
    }
  ]
}
```

## Build

```bash
go build -o infisicalservice ./cmd/
```

## Run

```bash
export INFISICAL_CLIENT_ID="your-client-id"
export INFISICAL_CLIENT_SECRET="your-client-secret"
PORT=8093 ./infisicalservice
```

## Testing

```bash
# Health check
curl http://localhost:8093/health

# Retrieve secrets (replace with your project ID)
curl -X POST http://localhost:8093/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d '{
    "@type": "RetrieveAction",
    "target": {
      "@type": "Project",
      "identifier": "your-project-id",
      "environment": "dev",
      "url": "https://app.infisical.com"
    }
  }'
```

## Security

- Secrets are masked in logs (only first 2 and last 2 characters shown)
- Credentials are stored as environment variables on scheduler host
- Optional API key authentication for production
- All communication over HTTPS when using external Infisical instance
- No secrets stored in workflow files (only project IDs and environments)

## Dependencies

- EVE v0.0.19 (Infisical semantic types)
- Echo v4 (HTTP framework)
- Infisical Go SDK v0.5.100

## License

Same as EVE library
