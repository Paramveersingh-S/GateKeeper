FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build consumer
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/consumer ./cmd/usage-consumer/main.go

# Minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

COPY --from=builder /bin/consumer /consumer

ENTRYPOINT ["/consumer"]
