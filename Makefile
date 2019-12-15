BINARY_NAME=go-infrabin

all: dep test

test:
	go test -v -race ./...

test-ci:
	go test -v -covermode=count -coverprofile=coverage.out ./...
	${HOME}/gopath/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken ${COVERALLS_TOKEN}

build:
	go build -o $(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

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
