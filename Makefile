.PHONY: all rust go clean

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
