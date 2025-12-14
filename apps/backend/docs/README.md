# API Documentation Pipeline

This document describes how API documentation is generated for the Drexa backend.

## Overview

The documentation pipeline converts Go code annotations into OpenAPI 3.0 specification that powers the Scalar API documentation UI.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Go Code +     │     │  Swagger 2.0    │     │  OpenAPI 3.0    │     │  OpenAPI 3.0    │
│ swag annotations│ ──▶ │  (swagger.json) │ ──▶ │  (converted)    │ ──▶ │  (patched)      │
└─────────────────┘     └─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │                       │
      swag                swagger2openapi        patch-openapi.ts         cmd/docs/
                                                                         openapi.json
```

## Why This Pipeline?

**The problem:** We want OpenAPI 3.0 features (like `oneOf` for union types) but the best annotation-based doc generator for Go (swag) only outputs Swagger 2.0.

**The solution:** A three-step pipeline:
1. Generate Swagger 2.0 with swag (keeps our existing handlers and annotations)
2. Convert to OpenAPI 3.0 with swagger2openapi
3. Patch in `oneOf` discriminated unions where needed

## Commands

### Generate documentation
```bash
make docs
```

This runs:
1. `swag init` → generates `docs/swagger.json`
2. `bunx swagger2openapi` → converts to `cmd/docs/openapi.json`
3. `bun scripts/patch-openapi.ts` → patches oneOf types

### Start documentation server
```bash
make run-docs
```

Opens Scalar UI at http://localhost:8081

## Adding New Union Types

If you add a new API that returns a union type (like `FileInfo | DirectoryInfo`), you need to update `scripts/patch-openapi.ts` to patch the schema.

Example:
```typescript
if (schemas['your_package.YourResponseType']) {
  const response = schemas['your_package.YourResponseType'];
  if (response.properties?.items) {
    response.properties.items = {
      type: 'array',
      items: {
        oneOf: [
          { $ref: '#/components/schemas/TypeA' },
          { $ref: '#/components/schemas/TypeB' }
        ],
        discriminator: {
          propertyName: 'kind',
          mapping: {
            a: '#/components/schemas/TypeA',
            b: '#/components/schemas/TypeB'
          }
        }
      }
    };
  }
}
```

## Files

| File | Purpose |
|------|---------|
| `docs/swagger.json` | Generated Swagger 2.0 spec (intermediate) |
| `cmd/docs/openapi.json` | Final OpenAPI 3.0 spec (embedded in docs server) |
| `scripts/patch-openapi.ts` | Patches oneOf types into the spec |
| `cmd/docs/main.go` | Scalar documentation server |

## Swag Annotations

Documentation is written as Go comments. See [swag documentation](https://github.com/swaggo/swag) for syntax.

Example:
```go
// createDirectory creates a new directory
// @Summary Create directory
// @Description Create a new directory within a parent directory
// @Tags directories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param accountID path string true "Account ID"
// @Param request body createDirectoryRequest true "Directory details"
// @Success 200 {object} DirectoryInfo "Created directory"
// @Failure 400 {object} map[string]string "Bad request"
// @Router /accounts/{accountID}/directories [post]
func (h *HTTPHandler) createDirectory(c *fiber.Ctx) error {
```

