.PHONY: build build-docs build-all run run-docs docs install-tools fmt clean

# Build the API server
build:
	go build -o bin/drexa ./cmd/drexa

# Build the documentation server
build-docs:
	go build -o bin/drexa-docs ./cmd/docs

# Build all binaries
build-all: build build-docs

# Run the API server
run:
	go run ./cmd/drexa --config config.yaml

# Run the documentation server
run-docs:
	go run ./cmd/docs --port 8081 --api-url http://localhost:8080

# Generate API documentation
docs:
	@echo "Generating Swagger 2.0 documentation..."
	swag init -g cmd/drexa/main.go -o docs --parseDependency --parseInternal --outputTypes json
	@echo "Converting to OpenAPI 3.0..."
	bunx --bun swagger2openapi docs/swagger.json -o cmd/docs/openapi.json --patch
	@echo "Patching OpenAPI spec with oneOf types..."
	bun scripts/patch-openapi.ts cmd/docs/openapi.json
	@echo "Documentation generated in docs/ and cmd/docs/"
	@echo "Run 'make run-docs' to start the documentation server"

# Install development tools
install-tools:
	go install github.com/swaggo/swag/cmd/swag@latest

# Format and lint
fmt:
	go fmt ./...
	swag fmt

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f docs/swagger.json docs/swagger.yaml cmd/docs/openapi.json
