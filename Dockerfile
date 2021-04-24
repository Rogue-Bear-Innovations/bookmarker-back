FROM golang:1.16

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/app

# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY . .

# Download all the dependencies
RUN go mod tidy

# Install the package
RUN go build -v -o ./bookmark ./cmd/app/main.go

# This container exposes port 8080 to the outside world
EXPOSE 1323

# Run the executable
CMD ["./bookmark"]
