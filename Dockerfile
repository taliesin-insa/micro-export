# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
# Bad practice but anyway
FROM golang:latest AS builder

# Add Maintainer Info
LABEL maintainer="Zoe LEVREL"

# Dépendances nécessaires pour compiler le fichier protocole
RUN apt-get update
RUN apt-get install -y protobuf-compiler
RUN go get -u github.com/golang/protobuf/proto
RUN go get -u github.com/golang/protobuf/protoc-gen-go

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Define directory
ADD src /src
WORKDIR /src/micro-export

# Download dependancies (if you try to build your image without following lines you will see missing packages)
RUN go get -u github.com/gorilla/mux github.com/stretchr/testify/assert

RUN go get -u github.com/prometheus/client_golang/prometheus
RUN go get -u github.com/prometheus/client_golang/prometheus/promauto
RUN go get -u github.com/prometheus/client_golang/prometheus/promhttp

# Build all project statically (prevent some exec user process caused "no such file or directory" error)
ENV CGO_ENABLED=0
RUN go build -o main .

# Build the docker image from a lightest one (otherwise it weights more than 1Go)
FROM alpine:latest

# Expose port 22022 to the outside world
EXPOSE 22022

# Don't really know what this does
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy on the executive env
COPY --from=builder /src/micro-export/ .

# Command to run the executable
CMD ["./main"]

