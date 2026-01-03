.PHONY: all rust go clean install

all: rust go

# Build the Rust library
rust:
	cd rust && cargo build --release

# Build the Go binary (depends on Rust library)
go: rust
	CGO_ENABLED=1 go build -o jayz .

# Clean build artifacts
clean:
	cd rust && cargo clean
	rm -f jayz

# Run the application
run: all
	DYLD_LIBRARY_PATH=./rust/target/release ./jayz

# Install the binary using go install and copy the library
install: rust
	@echo "Installing Rust library to /usr/local/lib..."
	@sudo install -d /usr/local/lib
	@sudo install -m 755 rust/target/release/libjjbridge.dylib /usr/local/lib/libjjbridge.dylib
	@echo "Installing jayz binary..."
	@CGO_ENABLED=1 go install .
	@echo "Installation complete! You can now run 'jayz' from anywhere."
	@echo "Note: The library is installed to /usr/local/lib"
