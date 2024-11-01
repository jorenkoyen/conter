.PHONY: lint test bench

lint:
	golangci-lint run

test:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm coverage.out

bench:
	go test -bench=. -benchmem ./...