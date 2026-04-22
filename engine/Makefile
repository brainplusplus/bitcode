.PHONY: build cli dev test lint clean tidy

build:
	go build -o bin/engine cmd/engine/main.go

cli:
	go build -o bin/bitcode cmd/bitcode/main.go

dev:
	go run cmd/engine/main.go

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf bin/

tidy:
	go mod tidy
