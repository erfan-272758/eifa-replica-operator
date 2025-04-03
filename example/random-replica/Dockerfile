# Build the app binary
FROM golang:1.22 AS builder
WORKDIR /app

# Copy the go source
COPY cmd/main.go cmd/main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o app cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/app .
USER 65532:65532

ENTRYPOINT ["/app"]