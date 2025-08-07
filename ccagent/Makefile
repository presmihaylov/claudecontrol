.PHONY: run build clean build-prod test test-verbose lint lint-fix

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

lint:
	golines --max-len=120 --dry-run .
	golangci-lint run

lint-fix:
	goimports -w .
	golines --max-len=120 --write-output .
	golangci-lint run --fix
