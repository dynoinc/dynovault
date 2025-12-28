# Format and tidy
fmt:
    gofmt -s -w .
    go mod tidy

# Run linters
lint: fmt
    go vet ./...
    go tool golangci-lint run ./...

# Run all tests
test: lint
    go test -v -count=1 -race -shuffle=on ./...
    uv run feastle/test_feast.py

# Run load generator against local server
loadgen:
    go run cmd/loadgen/main.go
