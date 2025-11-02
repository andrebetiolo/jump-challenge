# Start from the official Golang image
FROM golang:1.24.5

# Set working directory
WORKDIR /app

# Copy go mod and go sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o app .

# Expose the port the app runs on
EXPOSE 7700

# Run the app
CMD ["./app"]