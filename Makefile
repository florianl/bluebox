.PHONY: build clean fmt lint test

build:
	go build

clean:
	go clean

fmt:
	go tool gofumpt -w .

lint:
	go tool staticcheck -checks=all -show-ignored -tests  ./...

test:
	go test ./...
