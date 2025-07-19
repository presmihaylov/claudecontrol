.PHONY: run build clean

run:
	go run cmd/main.go

build:
	@mkdir -p bin
	go build -o bin/ccbackend cmd/main.go

clean:
	rm -rf bin/