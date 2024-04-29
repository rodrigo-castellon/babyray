# baby🦈

## Automatic deployment

You can automatically deploy the worker + GCS + global scheduler nodes by doing:

```bash
docker-compose up
```

Reset by doing ^C and then running `docker-compose down` to make sure all the containers are killed.


## Run a single node

To run a single Ray node, just do `./run.sh`. This will put you into a shell, and then you can run something like:

```bash
./go/bin/worker
```

and this will just launch the "worker" Go service. You should see something like:

```
root@fc8df146bfc1:/app# ./go/bin/worker
2024/04/27 00:55:59 server listening at [::]:50002
```

Remember to do ^D to exit the Docker container interactive session.


## Getting Started (manual)

Install everything you need first.

```
brew install go
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
pip install grpcio-tools
```


Then, run:
```
make all
```

This will compile all the Go server files and generate the Python/Go gRPC stub code.

Then, run
```
./go/bin/driver
```

This will start up the driver server.

Finally, run:

```
python3 python/babyray/client.py
```

This will run the Python client that will talk to the driver server.

## Info for contributors

Set up some global naming standards so that we can all follow along.

We assign each node a non-negative integer:
- 0: GCS
- 1: global scheduler
- 2 and above: worker nodes

We take advantage of DNS to avoid having to address nodes by their IP addresses. Specifically, name each container a separate name, "nodeX" where "X" is the node's integer ID. For example, "node0" for GCS.

Standard Docker allows us to do this: create a network with `docker network create mynetwork` and then any time you do `docker run` you add `--network mynetwork --network-alias myname` (replace `myname` with the DNS name). Also works in Docker Compose. So for now, just assume these names are there and Go's gRPC library will handle DNS resolution + the actual communication under the hood.

We also decide on port numbers for services. For local worker nodes:
- 50000: local object store
- 50001: local scheduler
- 50002: 0th worker
- 50003: 1st worker
- 50004: 2nd worker
- ...

For GCS:
- 50000: function table service
- 50001: object table service

For the global scheduler: port 50000.

## Using the Python package

Go to the `python/` directory and do `python3 -m pip install -e .` (or just `pip install -e .`, depending on whether you like the Python executable that your pip is attached to).


Then, go to some arbitrary place on your computer, and you can run this script with Python.
```
import babyray
babyray.init()  # initialize in the same way as the real Ray does

@babyray.remote
def f(x):
    return x * x

futures = [f.remote(i) for i in range(4)]
print(babyray.get(futures))  # [0, 1, 4, 9]
```

It doesn't work right now because it hits an error when it tries to query the local scheduler (which doesn't exist),
but this will start working once we implement all the Go stuff.


## File structure

file structure is kinda like:
```
babyray/
│
├── docs/                   # Documentation files
│   ├── setup.md
│   └── usage.md
│
├── python/                 # Python client package
│   ├── babyray/            # Source code for the babyray package
│   │   ├── __init__.py
│   │   ├── client.py       # gRPC client implementation
│   │   └── utils.py        # Helper functions and utilities
│   │
│   ├── tests/              # Unit tests for Python code
│   │   ├── __init__.py
│   │   ├── test_client.py
│   │   └── test_utils.py
│   │
│   ├── setup.py            # Setup script for the babyray package
│   └── requirements.txt    # Python dependencies
│
├── go/                     # Go server-side implementation
│   ├── cmd/                # Main applications for the project
│   │   ├── server/         # Server application entry point
│   │   │   └── main.go
│   │   └── client/         # Example Go client, if applicable
│   │       └── main.go
│   │
│   ├── pkg/                # Library code that's ok to use by external applications
│   │   ├── objectstore/    # Implementation of the object store
│   │   │   ├── server.go   # Server-side implementations
│   │   │   └── client.go   # Client-side implementations (if needed)
│   │   └── core/           # Core shared libraries
│   │
│   ├── internal/           # Private application and library code
│   │   └── config/         # Configuration related functionality
│   │
│   └── go.mod              # Go module definitions
│   └── go.sum              # Go module checksums
│
├── proto/                  # Protocol Buffers definitions
│   ├── ray.proto           # gRPC service definitions
│   └── Makefile            # Automate proto generation for Python and Go
│
├── scripts/                # Scripts for various build, run, or deployment tasks
│   ├── deploy.sh
│   └── setup_env.sh
│
├── .gitignore              # Specifies intentionally untracked files to ignore
├── LICENSE                 # License file
└── README.md               # Project overview and setup instructions
```

thanks to chatgpt for this. https://chat.openai.com/share/65e914c1-a030-45c7-9019-e7647d9707c2
