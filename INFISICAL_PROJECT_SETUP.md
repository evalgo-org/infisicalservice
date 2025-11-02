# Infisical Project Setup Guide

Complete reference for setting up Infisical projects for the IQS workflow system.

## Quick Reference

### Required Infisical Projects

| Project ID | Environments | Secret Count | Purpose |
|------------|--------------|--------------|---------|
| `iqs-s3-secrets` | dev, prod | 3 | Hetzner S3 storage credentials |
| `iqs-poolparty-secrets` | dev, prod | 4 | PoolParty SPARQL endpoint auth |
| `iqs-basex-secrets` | dev, prod | 4 | BaseX database credentials |

---

## Project 1: iqs-s3-secrets

**Purpose**: Hetzner S3 object storage for IQS cache files

**Environments**: `dev`, `prod`

### Secrets to Create:

| Secret Name | Example Value | Description |
|------------|---------------|-------------|
| `HETZNER_S3_URL` | `https://fsn1.your-objectstorage.com` | S3-compatible endpoint URL |
| `HETZNER_S3_ACCESS_KEY` | `4JDEMZEQ4XHNCSMCTIY7` | S3 access key ID |
| `HETZNER_S3_SECRET_KEY` | `BdQ9Gr4QVBo4YxgY4ov5n1DZA6GXBeIBQmfqD6JO` | S3 secret access key |

**Optional**:
- `HETZNER_S3_REGION` (default: `fsn1`)
- `HETZNER_S3_BUCKET` (default: `px-semantic`)

**Used in workflows**:
- iqs-cache-prepare-workflow.json (positions 2, 8, 10)
- iqs-cache-user-certificate-workflow.json (position 8)
- iqs-cache-empolis-jsons-workflow.json (position 9)
- iqs-cache-top-concepts-workflow.json
- iqs-cache-schema-st4-workflow.json

**Example RetrieveAction**:
```json
{
  "@type": "RetrieveAction",
  "@id": "#retrieve-s3-secrets",
  "identifier": "get-s3-credentials",
  "target": {
    "@type": "Project",
    "identifier": "iqs-s3-secrets",
    "environment": "prod",
    "url": "https://app.infisical.com"
  }
}
```

---

## Project 2: iqs-poolparty-secrets

**Purpose**: PoolParty SPARQL endpoint authentication

**Environments**: `dev`, `prod`

### Secrets to Create:

| Secret Name | Example Value | Description |
|------------|---------------|-------------|
| `POOLPARTY_SPARQL_URL` | `https://zeiss-prod.poolparty.biz/PoolParty/sparql/IQS` | SPARQL endpoint URL |
| `POOLPARTY_SPARQL_USERNAME` | `iqs_service_account` | SPARQL authentication username |
| `POOLPARTY_SPARQL_PASSWORD` | `your-secure-password` | SPARQL authentication password |
| `POOLPARTY_PROJECT_ID` | `IQS` | PoolParty project identifier |

**Used in workflows**:
- All workflows that execute SPARQL queries (SearchAction)
- Currently hardcoded in workflows (needs migration to secrets)

**Example RetrieveAction**:
```json
{
  "@type": "RetrieveAction",
  "@id": "#retrieve-poolparty-secrets",
  "identifier": "get-poolparty-credentials",
  "target": {
    "@type": "Project",
    "identifier": "iqs-poolparty-secrets",
    "environment": "prod",
    "url": "https://app.infisical.com"
  }
}
```

---

## Project 3: iqs-basex-secrets

**Purpose**: BaseX XML database credentials for XQuery transformations

**Environments**: `dev`, `prod`

### Secrets to Create:

| Secret Name | Example Value | Description |
|------------|---------------|-------------|
| `BASEX_URL` | `http://localhost:8090` | BaseX service endpoint |
| `BASEX_USERNAME` | `admin` | BaseX admin username |
| `BASEX_PASSWORD` | `your-basex-password` | BaseX admin password |
| `BASEX_DATABASE` | `iqs_cache` | Default database name |

**Used in workflows**:
- All workflows that perform XQuery transformations (TransformAction, UploadAction)
- Currently using localhost:8090 (authentication may be added for production)

**Example RetrieveAction**:
```json
{
  "@type": "RetrieveAction",
  "@id": "#retrieve-basex-secrets",
  "identifier": "get-basex-credentials",
  "target": {
    "@type": "Project",
    "identifier": "iqs-basex-secrets",
    "environment": "prod",
    "url": "https://app.infisical.com"
  }
}
```

---

## Scheduler Host Environment Variables

These are **NOT** stored in Infisical. Store these as environment variables on the when/fetcher host:

```bash
# Required for infisicalservice to authenticate with Infisical
export INFISICAL_CLIENT_ID="your-universal-auth-client-id"
export INFISICAL_CLIENT_SECRET="your-universal-auth-client-secret"

# Optional: API key protection for infisicalservice endpoint
export INFISICAL_SERVICE_API_KEY="optional-api-key-for-service-protection"
```

---

## Setup Steps

### 1. Create Infisical Projects

In your Infisical instance (https://app.infisical.com):

1. Create project: `iqs-s3-secrets`
   - Add environment: `dev`
   - Add environment: `prod`
   - Add 3 secrets (see table above)

2. Create project: `iqs-poolparty-secrets`
   - Add environment: `dev`
   - Add environment: `prod`
   - Add 4 secrets (see table above)

3. Create project: `iqs-basex-secrets`
   - Add environment: `dev`
   - Add environment: `prod`
   - Add 4 secrets (see table above)

### 2. Create Universal Auth Credentials

In Infisical:

1. Navigate to Project Settings → Access Control → Machine Identities
2. Create new Machine Identity: `iqs-workflow-executor`
3. Generate Universal Auth credentials
4. Copy Client ID and Client Secret
5. Grant access to all three projects created above

### 3. Configure Scheduler Host

On the when/fetcher host:

```bash
# Create .env file or add to shell profile
cat > /home/opunix/when/.env <<'EOF'
export INFISICAL_CLIENT_ID="your-client-id-from-step-2"
export INFISICAL_CLIENT_SECRET="your-client-secret-from-step-2"
export INFISICAL_SERVICE_API_KEY="optional-production-api-key"
EOF

# Load environment
source /home/opunix/when/.env
```

### 4. Start infisicalservice

```bash
cd /home/opunix/infisicalservice
PORT=8093 ./infisicalservice
```

Verify it's running:
```bash
curl http://localhost:8093/health
```

### 5. Test Secret Retrieval

Test retrieving S3 secrets:

```bash
curl -X POST http://localhost:8093/v1/api/semantic/action \
  -H "Content-Type: application/json" \
  -d '{
    "@type": "RetrieveAction",
    "target": {
      "@type": "Project",
      "identifier": "iqs-s3-secrets",
      "environment": "dev",
      "url": "https://app.infisical.com"
    }
  }'
```

---

## Migration Guide: Update Existing Workflows

Current workflows have hardcoded credentials. Here's how to migrate them:

### Before (Hardcoded):
```json
{
  "@type": "CreateAction",
  "target": {
    "@type": "DataCatalog",
    "url": "https://fsn1.your-objectstorage.com",
    "additionalProperty": {
      "accessKey": "4JDEMZEQ4XHNCSMCTIY7",
      "secretKey": "BdQ9Gr4QVBo4YxgY4ov5n1DZA6GXBeIBQmfqD6JO"
    }
  }
}
```

### After (With Secrets):
```json
{
  "@type": "ItemList",
  "itemListElement": [
    {
      "position": 1,
      "item": {
        "@type": "RetrieveAction",
        "@id": "#retrieve-s3-secrets",
        "target": {
          "@type": "Project",
          "identifier": "iqs-s3-secrets",
          "environment": "prod",
          "url": "https://app.infisical.com"
        }
      }
    },
    {
      "position": 2,
      "item": {
        "@type": "CreateAction",
        "additionalProperty": {"dependsOn": "#retrieve-s3-secrets"},
        "target": {
          "@type": "DataCatalog",
          "url": "${secrets.HETZNER_S3_URL}",
          "additionalProperty": {
            "accessKey": "${secrets.HETZNER_S3_ACCESS_KEY}",
            "secretKey": "${secrets.HETZNER_S3_SECRET_KEY}"
          }
        }
      }
    }
  ]
}
```

---

## Troubleshooting

### Issue: "Infisical authentication failed"

**Solution**: Verify INFISICAL_CLIENT_ID and INFISICAL_CLIENT_SECRET are correct
```bash
env | grep INFISICAL
```

### Issue: "Secret XYZ not found"

**Solution**: Verify secret name matches exactly (case-sensitive):
- Check in Infisical web UI
- Ensure environment (dev/prod) is correct
- Verify project identifier matches

### Issue: "infisicalservice not responding"

**Solution**: Check service is running:
```bash
lsof -i :8093
curl http://localhost:8093/health
```

---

## Security Best Practices

1. ✅ **Never commit secrets to git** - use Infisical for all credentials
2. ✅ **Use separate environments** - dev and prod with different credentials
3. ✅ **Rotate credentials regularly** - update in Infisical, not in code
4. ✅ **Limit access** - use Infisical's access control for team members
5. ✅ **Enable API key** - set INFISICAL_SERVICE_API_KEY in production
6. ✅ **Monitor logs** - secrets are automatically masked in infisicalservice logs
7. ✅ **Use HTTPS** - ensure Infisical instance uses HTTPS (app.infisical.com does)

---

## Summary

**Total secrets to configure**: 11 secrets across 3 projects

**Time to setup**: ~15 minutes

**Benefits**:
- No hardcoded credentials in workflow files
- Centralized secret management
- Easy credential rotation
- Audit trail of secret access
- Environment isolation (dev/prod)

**Next steps after setup**:
1. Test each project's secret retrieval
2. Update workflows to use RetrieveAction pattern
3. Remove hardcoded credentials from existing workflows
4. Test end-to-end workflow execution
