FROM golang:1.13.4-stretch AS builder
WORKDIR /src
COPY server.go cli.go go.mod /src/
RUN go install ./...

FROM scratch
COPY --from=builder /go/bin/otre /otre
EXPOSE 9411
CMD ["/otre", "--port", "9411"]
