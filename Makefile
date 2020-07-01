BINARY_NAME=go-infrabin

all: dep test

lint:
	golangci-lint run

test: lint
	go test -v -race ./...

test-ci:
	go test -v -covermode=count -coverprofile=coverage.out ./...
	${HOME}/gopath/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken ${COVERALLS_TOKEN}

protoc:
	protoc --proto_path=proto/ --go_out=plugins=grpc:pkg/infrabin --go_opt=paths=source_relative proto/infrabin.proto

build: protoc
	go build -o $(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

run: build
	./$(BINARY_NAME)

dep:
	go get ./...

dep-ci: dep
	go get golang.org/x/tools/cmd/cover
	go get github.com/mattn/goveralls

# Clean go.mod
go-mod-tidy:
	@go mod tidy -v
	@git diff HEAD
	@git diff-index --quiet HEAD

clean:
	rm -f $(BINARY_NAME)
