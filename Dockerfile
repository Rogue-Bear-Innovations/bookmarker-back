FROM golang:1.16

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/app

# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY . .

RUN go mod tidy

RUN go build -v -o ./bookmark ./cmd/app/main.go

# Run the executable
CMD ["./bookmark"]
