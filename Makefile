.PHONY: run build clean dev test vendor-check vendor-update

# Default target
all: run

# Run the application in development mode
run:
	go run cmd/server/main.go

# Build the application
build:
	go build -o bin/gote cmd/server/main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf data/

# Check vendor library health
vendor-check:
	@echo "🔍 Checking vendor libraries..."
	@npm run check-vendor

# Update vendor libraries
vendor-update:
	@echo "📦 Updating vendor libraries..."
	@npm run update-vendor

# Development mode with automatic restart (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run mode..."; \
		go run main.go; \
	fi

# Run tests (when we add them)
test:
	go test ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Create a sample note for testing
sample:
	@mkdir -p data
	@echo '{"id":1,"title":"Welcome to Meemo","content":"This is your first note!\n\nYou can:\n- Create new notes\n- Search through your notes\n- Edit and delete notes\n- Use keyboard shortcuts (Ctrl+S to save, Ctrl+N for new note)\n\nEnjoy taking notes!","created_at":"2024-01-15T10:00:00Z","updated_at":"2024-01-15T10:00:00Z"}' > data/1.json
	@echo "Sample note created in data/1.json"

# Show help
help:
	@echo "Available targets:"
	@echo "  run     - Run the application"
	@echo "  build   - Build the application binary"
	@echo "  clean   - Clean build artifacts and data"
	@echo "  dev     - Run in development mode (with air if available)"
	@echo "  test    - Run tests"
	@echo "  deps    - Install/update dependencies"
	@echo "  sample  - Create a sample note for testing"
	@echo "  help    - Show this help message"
