.PHONY: build clean

run:
	go run cmd/*.go

build:
	mkdir -p bin
	go build -o bin/ccagent cmd/main.go

clean:
	rm -rf bin/
