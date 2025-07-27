.PHONY: build clean build-prod

run:
	go run cmd/*.go --verbose

build:
	mkdir -p bin
	go build -o bin/ccagent cmd/*.go

clean:
	rm -rf bin/

build-prod:
	./scripts/build_binaries.sh
