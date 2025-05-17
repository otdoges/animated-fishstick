# Build stage
FROM golang:1.20-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./
# Copy go.sum if it exists
COPY go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o supabase-go ./cmd/api

# Final stage
FROM alpine:3.18

# Install necessary runtime packages
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/supabase-go .

# Copy embedded files (migrations, etc.)
COPY --from=builder /app/db/migrations ./db/migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["./supabase-go"]
