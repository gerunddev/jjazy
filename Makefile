.PHONY: all rust go clean install release test lint fmt

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-s -w -X main.Version=$(VERSION)"

all: rust go

# Build the Rust static library
rust:
	cd rust && cargo build --release

# Build the Go binary (statically linked with Rust)
go: rust
	CGO_ENABLED=1 go build $(LDFLAGS) -o jjazy .

# Build optimized release binary with verification
release: rust
	CGO_ENABLED=1 go build $(LDFLAGS) -o jjazy .
	@echo "Binary size: $$(du -h jjazy | cut -f1)"
	@echo "Verifying no dynamic Rust dependencies..."
	@otool -L jjazy | grep -v libjjbridge || echo "OK: No libjjbridge.dylib dependency"

# Clean build artifacts
clean:
	cd rust && cargo clean
	rm -f jjazy

# Run the application (no DYLD_LIBRARY_PATH needed!)
run: all
	JJAZY_LOG_FILE=/tmp/jjazy.log JJAZY_LOG_LEVEL=debug ./jjazy

# Install the binary to /usr/local/bin
install: release
	@echo "Installing jjazy to /usr/local/bin..."
	@sudo install -m 755 jjazy /usr/local/bin/jjazy
	@echo "Installation complete! Run 'jjazy' from any jj repository."

# Run tests
test: rust
	CGO_ENABLED=1 go test ./...

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -w .
