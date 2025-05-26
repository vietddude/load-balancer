FROM golang:1.22-alpine

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Generate go.sum and download dependencies
RUN go mod tidy

# Copy source code
COPY . .

# Build the application
RUN go build -o main ./cmd/loadbalancer

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./main"] 