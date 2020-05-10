FROM golang:1.14.2-buster as builder
WORKDIR /go/src/go-infrabin
COPY . /go/src/go-infrabin
RUN make build

FROM gcr.io/distroless/base-debian10
COPY --from=builder /go/src/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
ENTRYPOINT [ "/usr/local/bin/go-infrabin" ]
