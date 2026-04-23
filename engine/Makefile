.PHONY: build cli dev test lint clean tidy

build:
	go build -o bin/engine ./cmd/engine/

cli:
	go build -o bin/bitcode ./cmd/bitcode/

dev:
	go run ./cmd/engine/

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf bin/

tidy:
	go mod tidy
