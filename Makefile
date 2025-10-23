BINARY_NAME=go-infrabin
GO_TEST_FLAGS ?= -v -covermode=atomic -coverprofile=coverage.out
GO_TEST_RACE_FLAG ?= -race

all: dep test

.PHONY: tools
tools:
	go generate -tags tools tools/tools.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

lint: fmt vet

test: protoc lint
	go test $(GO_TEST_FLAGS) $(GO_TEST_RACE_FLAG) ./...

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
