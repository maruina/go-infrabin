BINARY_NAME=go-infrabin

all: dep test

test:
	go test -v -race ./...

build:
	go build -o $(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

dep:
	go get ./...

clean:
	rm -f $(BINARY_NAME)