# Stage 1: Build the application
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./main.go

# Stage 2: Create the final, lightweight image
FROM alpine:latest
 
# Create a non-root user to run the application for better security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
 
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /main .

# Change ownership of the app directory to our non-root user
RUN chown -R appuser:appgroup /app
USER appuser

# Command to run the application
CMD ["./main"]
