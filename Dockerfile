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


# now that we've installed pre-reqs, build everything
RUN make clean && make all

# just to test things out
RUN apt update && apt install -y iputils-ping

ENV PROJECT_ROOT=/app

# expose all the ports we may use
#EXPOSE 50000-69999

# placeholder commands
#CMD ["./bin/localobjstore", "&", "./bin/localscheduler", "&", "./bin/worker"]

