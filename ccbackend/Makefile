.PHONY: run build clean test test-verbose

run:
	go run cmd/main.go

build:
	@mkdir -p bin
	go build -o bin/ccbackend cmd/*.go

clean:
	rm -rf bin/

test:
	go test -count=1 ./...

test-verbose:
	go test -count=1 ./... -v