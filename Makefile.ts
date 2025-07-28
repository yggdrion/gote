# Makefile for Gote TypeScript/Bun version

.PHONY: dev build start install clean test vendor-update vendor-check

# Default target
all: install build

# Install dependencies
install:
	bun install

# Development server with hot reload
dev:
	bun run dev

# Build for production
build:
	bun run build

# Start production server
start:
	bun run start

# Clean build artifacts
clean:
	rm -rf dist/
	rm -rf node_modules/

# Update vendor dependencies
vendor-update:
	bun run update-vendor

# Check vendor dependencies
vendor-check:
	bun run check-vendor

# Run type checking
typecheck:
	bun run --bun tsc --noEmit

# Format code
format:
	bun run --bun prettier --write src/

# Help
help:
	@echo "Available targets:"
	@echo "  install      - Install dependencies"
	@echo "  dev          - Start development server"
	@echo "  build        - Build for production"
	@echo "  start        - Start production server"
	@echo "  clean        - Clean build artifacts"
	@echo "  vendor-update - Update vendor dependencies"
	@echo "  vendor-check  - Check vendor dependencies"
	@echo "  typecheck    - Run TypeScript type checking"
	@echo "  format       - Format code with Prettier"
