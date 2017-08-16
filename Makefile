COMMIT=$(shell git rev-parse HEAD 2> /dev/null || true)

default: tool

tool: 
	go build -ldflags "-X main.gitCommit=${COMMIT}" -o docker2oci .

update:
	vndr

