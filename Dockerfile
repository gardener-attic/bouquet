### Build Stage ###
FROM golang:1.10.3 AS build

WORKDIR /go/src/github.com/gardener/bouquet
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bouquet cmd/main.go

### Controller  ###
FROM alpine:3.7

COPY --from=build /bouquet /bouquet

ENTRYPOINT ["/bouquet"]
