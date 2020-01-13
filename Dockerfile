FROM golang:1.13.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY cli.go server.go forwarder.go /src/
COPY traces/ /src/traces/
COPY rules/ /src/rules/

RUN go build ./... && go test ./... && go install ./...

FROM alpine:3.10
COPY --from=builder /go/bin/otre /go/bin/otre
EXPOSE 9411
CMD ["/go/bin/otre", "--port", "9411"]
