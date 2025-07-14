COMMIT=$(shell git rev-parse HEAD 2> /dev/null || true)

# Build flags for optimization
LDFLAGS = -X main.gitCommit=${COMMIT} -s -w
GCFLAGS = -m -l
BUILDFLAGS = -a -installsuffix cgo -trimpath

default: tool

tool: 
	go build -ldflags "${LDFLAGS}" -gcflags "${GCFLAGS}" ${BUILDFLAGS} -o docker2oci .

# Optimized build with additional size reduction
optimized:
	go build -ldflags "${LDFLAGS}" -gcflags "${GCFLAGS}" ${BUILDFLAGS} -o docker2oci .
	strip docker2oci 2>/dev/null || true
	upx --lzma docker2oci 2>/dev/null || true

# Development build with debug info
dev:
	go build -ldflags "-X main.gitCommit=${COMMIT}" -o docker2oci .

# Clean build artifacts
clean:
	rm -f docker2oci

# Install dependencies
install-deps:
	go mod download
	go mod verify

# Run tests
test:
	go test -v ./...

# Benchmark
bench:
	go test -bench=. -benchmem

update:
	vndr

