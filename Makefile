VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
BINARY  := spekk
SRC     := ./cmd/spekk
DIST    := dist

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: build build-all clean

# Build for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(SRC)

# Cross-compile for all supported platforms
build-all:
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(DIST)/$(BINARY)-$${os}-$${arch}; \
		if [ "$$os" = "windows" ]; then output=$${output}.exe; fi; \
		echo "Building $$output ..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$output $(SRC) || exit 1; \
	done
	@echo "All binaries built in $(DIST)/"

clean:
	rm -rf $(DIST)
