# Start from a base Go image
FROM golang:latest 

# Set the working directory inside the container
WORKDIR /app

# Copy the Go application source code into the container
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port your application listens on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
