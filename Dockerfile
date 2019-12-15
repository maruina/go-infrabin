FROM golang:1.13.5-buster as builder
WORKDIR /go/src/go-infrabin
ADD . /go/src/go-infrabin
RUN make build

FROM gcr.io/distroless/base-debian10
COPY --from=builder /go/src/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
ENTRYPOINT [ "/usr/local/bin/go-infrabin" ]
