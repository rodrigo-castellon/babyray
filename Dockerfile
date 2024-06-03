# Declare the CONFIG argument at the beginning
ARG CONFIG=base

# Base stage for common setup
FROM golang as base

# Install gcc and other necessary tools
# we need gcc for Golang SQLite bc uses C code
RUN apt-get update && \
    apt-get install -y gcc && \ 
    apt-get install -y protobuf-compiler && \
    apt-get clean

# Set CGO_ENABLED=1 , we need this for Golang SQLite bc uses C code
ENV CGO_ENABLED=1

# Set the Current Working Directory inside the container
WORKDIR /app


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

# just to test things out
RUN apt update && apt install -y iputils-ping


# Create the required directory structure
RUN mkdir -p go

# Copy the go.mod and go.sum files into the go directory
COPY go/go.mod go/go.sum go/

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN cd go && go mod download

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# now that we've installed pre-reqs, build everything
RUN make clean && make go && make py && make build

ENV PROJECT_ROOT=/app

FROM base as driver

# install necessary Python packages to run anything
RUN python3 -m pip install dill cloudpickle --break-system-packages
RUN cd python && python3 -m pip install -e . --break-system-packages

# install basic necessities to actually do driver stuff
RUN apt install -y nano

# install testing
RUN python3 -m pip install pytest --break-system-packages

# take in a CONFIG argument which will tell us what to target (GCS, global scheduler, or worker)
# using multi-stage builds: https://chat.openai.com/share/a5eb4076-e36a-4a1e-b4c8-9d56ea7a604e
FROM ${CONFIG} as final
