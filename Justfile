default:
  go run cmd/dotbrain/main.go
format:
  go fmt
test:
  go test ./...
help:
  @echo "Available commands:"
  @echo "  default: Run the main application."
  @echo "  help: Show this help message."
