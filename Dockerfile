FROM golang:1.13.5-buster as builder
WORKDIR /go/src/github.com/maruina/go-infrabin
COPY . .
RUN go build -o go-infrabin cmd/go-infrabin/main.go

FROM gcr.io/distroless/base-debian10
COPY --from=builder /go/src/github.com/maruina/go-infrabin/go-infrabin /usr/local/bin/go-infrabin
ENTRYPOINT [ "/usr/local/bin/go-infrabin" ]
