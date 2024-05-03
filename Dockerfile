# Start with the official Golang base image for the build stage
FROM golang:1.18 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files to the container
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files do not change
RUN go mod download

# Copy the source code into the container
COPY . .

COPY index.html .

# Build the application. Disable CGO and target Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Start a new stage from scratch for the runtime
FROM alpine:latest

# Set the working directory in the container
WORKDIR /root/

# Copy the built executable from the builder stage to the production image
COPY --from=builder /app/main .

# Expose port 8080 for the application
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
