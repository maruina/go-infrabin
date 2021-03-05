BINARY_NAME=go-infrabin

all: dep test

lint:
	golangci-lint run

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

lint: fmt vet

test: lint
	go test -v -covermode=atomic -coverprofile=coverage.out -race ./...

protoc:
	protoc \
		--proto_path=proto/ \
		--go_out=paths=source_relative:pkg \
		--go-grpc_out=paths=source_relative:pkg \
		--grpc-gateway_out=logtostderr=true,paths=source_relative:pkg \
		proto/infrabin/infrabin.proto
	protoc \
        --proto_path=proto/ \
        --grpc-gateway_out=logtostderr=true,paths=source_relative,standalone=true:pkg \
        proto/grpc/health/v1/health.proto

build: protoc
	go build -o $(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

run: build
	./$(BINARY_NAME)

dep:
	go get ./...

dep-ci: dep
	go get golang.org/x/tools/cmd/cover
	go get github.com/mattn/goveralls

clean:
	rm -f $(BINARY_NAME)
