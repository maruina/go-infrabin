BINARY_NAME=go-infrabin

test:
	go test -v -race ./...

build:
	go build -o $(BINARY_NAME) cmd/main.go

clean:
	rm -f $(BINARY_NAME)