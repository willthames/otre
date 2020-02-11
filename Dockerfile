FROM golang:1.13.6-alpine3.11 AS builder
WORKDIR /src
RUN apk update && apk add git
COPY go.mod go.sum /src/
RUN go mod download
COPY server.go /src/
COPY traces/ /src/traces/
COPY rules/ /src/rules/

ENV CGO_ENABLED 0
RUN go build ./... && go test ./... && go install ./...

FROM alpine:3.11
COPY --from=builder /go/bin/otre /go/bin/otre
EXPOSE 9411
CMD ["/go/bin/otre", "--port", "9411"]
