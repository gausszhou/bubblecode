.PHONY: build test clean install build-all build-linux build-darwin build-windows lint fmt vet package

BINARY_NAME=bubblecode
BIN_DIR=bin
DIST_DIR=dist
VERSION?=dev

build: build-all

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

install:
	@echo "Installing $(BINARY_NAME)..."
	CGO_ENABLED=0 go install -ldflags="-s -w" .
	@echo "$(BINARY_NAME) installed successfully to $$(go env GOPATH)/bin"

build-all: build-linux build-darwin build-windows

build-linux:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 .

build-darwin:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .

build-windows:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe .

package: build-all
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	# Linux
	tar czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-linux-amd64
	tar czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-linux-arm64
	# Darwin
	tar czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-darwin-amd64
	tar czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(BIN_DIR) $(BINARY_NAME)-darwin-arm64
	# Windows (zip)
	@powershell -Command "Compress-Archive -Path $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe -DestinationPath $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip -Force"
	@powershell -Command "Compress-Archive -Path $(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe -DestinationPath $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-arm64.zip -Force"

lint:
	@which golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...