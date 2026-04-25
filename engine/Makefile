.PHONY: build install dev serve test lint clean tidy

build:
	go build -o bin/bitcode ./cmd/bitcode/

install:
	go install ./cmd/bitcode/

serve:
	go run ./cmd/bitcode/ serve

dev:
	go run ./cmd/bitcode/ dev

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -rf bin/ tmp/

tidy:
	go mod tidy
