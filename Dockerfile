FROM golang:1.13.4-stretch AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY cli.go server.go /src/
COPY traces/ /src/traces/
COPY rules/ /src/rules/

RUN go build ./... && go test ./... && go install ./...

EXPOSE 9411
CMD ["/go/bin/otre", "--port", "9411"]
