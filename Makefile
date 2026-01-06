.PHONY: build install clean test lint run

BINARY_NAME=gh-repo-review

build:
	go build -o $(BINARY_NAME) .

install: build
	gh extension install .

uninstall:
	gh extension remove repo-review

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test -v ./...

lint:
	golangci-lint run

run: build
	./$(BINARY_NAME)

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Development with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air
