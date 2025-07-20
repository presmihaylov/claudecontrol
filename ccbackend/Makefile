.PHONY: run build clean test test-verbose

run:
	go run cmd/main.go

build:
	@mkdir -p bin
	go build -o bin/ccbackend cmd/*.go

clean:
	rm -rf bin/

test:
	go test ./...

test-verbose:
	go test ./... -v