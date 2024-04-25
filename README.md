# baby🦈

## Getting Started

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


## Using the Python package

Go to the `python/` directory and do `python3 -m pip install -e .` (or just `pip install -e .`, depending on whether you like the Python executable that your pip is attached to).


Then, go to some arbitrary place on your computer, and you can run this script with Python and it should work.
```
import babyray
babyray.init()  # initialize in the same way as the real Ray does

@babyray.remote
def f(x):
    return x * x

futures = [f.remote(i) for i in range(4)]
print(babyray.get(futures))  # [0, 1, 4, 9]
```


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
