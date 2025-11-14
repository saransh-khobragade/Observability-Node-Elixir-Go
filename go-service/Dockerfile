FROM golang:1.21-alpine

# Install curl for healthcheck
RUN apk add --no-cache curl

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy application code
COPY . .

# Build the application
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]

