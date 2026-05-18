# syntax=docker/dockerfile:1.7

FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOFLAGS="-trimpath" \
    go build -ldflags="-s -w" -o /out/mcp-google-sheets ./

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/mcp-google-sheets /app/mcp-google-sheets
USER nonroot:nonroot
# SSE transport defaults to 127.0.0.1 in code. Override HOST=0.0.0.0 at runtime
# only when you understand that the MCP endpoint has no built-in auth.
EXPOSE 8000
ENTRYPOINT ["/app/mcp-google-sheets"]
CMD ["--transport", "sse"]
