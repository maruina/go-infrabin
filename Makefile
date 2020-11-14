BINARY_NAME=go-infrabin
GIT_COMMIT := $(shell git rev-parse --short HEAD)

all: dep test

lint:
	golangci-lint run

test: lint
	go test -v -race ./...

test-ci:
	go test -v -covermode=count -coverprofile=coverage.out ./...
	${HOME}/gopath/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken ${COVERALLS_TOKEN}

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
	sed \
		-i.bak \
		-e 's/PopulateQueryParameters(&protoReq/PopulateQueryParameters(protov1.MessageV2(\&protoReq)/g' \
		-e 's/msg, metadata, err/protov1.MessageV2(msg), metadata, err/g' \
		pkg/grpc/health/v1/health.pb.gw.go
	mv pkg/grpc/health/v1/health.pb.gw.go pkg/grpc/health/v1/health.pb.gw.go.bak
	head -n 23 pkg/grpc/health/v1/health.pb.gw.go.bak > pkg/grpc/health/v1/health.pb.gw.go
	echo '    protov1 "github.com/golang/protobuf/proto"' >> pkg/grpc/health/v1/health.pb.gw.go
	tail -n +24 pkg/grpc/health/v1/health.pb.gw.go.bak >> pkg/grpc/health/v1/health.pb.gw.go
	rm pkg/grpc/health/v1/health.pb.gw.go.bak

build: protoc
	go build -o $(BINARY_NAME) -ldflags "-X github.com/maruina/go-infrabin/cmd.gitCommit=$(GIT_COMMIT)" main.go

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
