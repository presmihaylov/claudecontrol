.PHONY: build clean build-prod test test-verbose

run:
	go run cmd/*.go

build:
	mkdir -p bin
	go build -o bin/ccagent cmd/*.go

clean:
	rm -rf bin/

build-prod:
	./scripts/build_binaries.sh

test:
	go test ./...

test-verbose:
	go test -v ./...
