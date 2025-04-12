build:
	@go build -o bin/gofs

run: build
	@./bin/gofs

test:
	@go test ./... -v