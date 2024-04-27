# Use the official Golang image to create a build artifact.
# This is based on Debian and includes the Go toolset.
FROM golang
#:1.19
#FROM golang:1.22
#FROM debian:buster
# as builder

RUN apt update
RUN apt install -y protobuf-compiler

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Install Python and pip.
# Debian's package manager is used to install Python and pip.
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

RUN ln -s /usr/bin/python3 /usr/bin/python

# Upgrade pip to its latest version.
RUN python3 -m pip install --upgrade pip --break-system-packages

# Install grpcio using pip.
RUN python3 -m pip install grpcio --break-system-packages

RUN python3 -m pip install grpcio-tools --break-system-packages

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2


# Build the Go app
# Assuming your go.mod file is in the same directory as your Dockerfile,
# otherwise, you'll need to adjust the paths accordingly.
#RUN make all
#RUN go build -o bin/worker cmd/worker/main.go
#RUN go build -o bin/localscheduler cmd/localscheduler/main.go
#RUN go build -o bin/localobjstore cmd/localobjstore/main.go

# Use a Docker multi-stage build to minimize the size of the final image by not including the build environment.
#FROM debian:buster-slim
#
## Set the Current Working Directory inside the container
#WORKDIR /app
#
## Copy the Pre-built binary file from the previous stage
#COPY --from=builder /app/bin /app/bin
#
## Install any needed packages specified in requirements.txt
## Install necessary packages
#RUN apt-get update && apt-get install -y \
#    ca-certificates \
#    && rm -rf /var/lib/apt/lists/*
#
## Expose ports (match the ports your apps are listening on)
#EXPOSE 50000 50001 50002
#
## Command to run the executable
## This CMD command starts your workers and scheduler. Adjust if your setup requires starting them differently.
#CMD ["./bin/localobjstore", "&", "./bin/localscheduler", "&", "./bin/worker"]

