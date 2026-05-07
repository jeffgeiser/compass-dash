BINARY   := compass-dash
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"
DIST_DIR := dist

.PHONY: build test release clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

release: test
	mkdir -p $(DIST_DIR)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe .

clean:
	rm -rf $(BINARY) $(DIST_DIR)
