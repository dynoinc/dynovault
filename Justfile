# Run feast integration test
feast-test:
    uv run feastle/test_feast.py

# Run load generator against local server
loadgen:
    go run cmd/loadgen/main.go
