.PHONY: run build clean test test-verbose lint lint-fix

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

lint:
	golines --max-len=120 --dry-run .
	golangci-lint run

lint-fix:
	golines --max-len=120 --write-output .
	golangci-lint run --fix
