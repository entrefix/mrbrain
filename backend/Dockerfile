# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Build migration script
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate-users ./scripts/migrate_users_to_supabase.go

# Build check users script
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o check-users ./scripts/check_users.go

# Build check memories script
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o check-memories ./scripts/check_memories.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests (OpenAI API) and sqlite3 for debugging
RUN apk --no-cache add ca-certificates tzdata sqlite

# Copy binaries from builder
COPY --from=builder /app/main .
COPY --from=builder /app/migrate-users .
COPY --from=builder /app/check-users .
COPY --from=builder /app/check-memories .

# Copy scripts for database management
COPY --from=builder /app/scripts /app/scripts
RUN chmod +x /app/scripts/*.sh

EXPOSE 8099

CMD ["./main"]
