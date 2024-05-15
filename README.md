# baby🦈

[Baby Ray](https://www.youtube.com/watch?v=WkCecpH2GAo) is a minimal yet fully-functioning (fault-tolerance included!) implementation of [Ray](https://arxiv.org/abs/1712.05889) in Go. 

Currently a work in progress. By the end of this project, you should be able to simulate a full Ray cluster using Docker Compose and launch CPU jobs using Python, using exactly the same API exposed by the real Ray Core library.

## Unit Tests

To run unit tests, either push or just run [act](https://github.com/nektos/act) locally.

## Integration Tests

To run integration tests:

```bash
make all
```

Then,

```bash
./scripts/run_tests.sh
```

This script will spin up a Baby Ray cluster and have the first worker node run a suite of Pytest tests (`tests/integration/test_cluster.py`).

## Automatic deployment - shorthand version
```bash
make all
```

```bash
docker-compose up
```

Separate terminal:
```bash
./log_into_driver.sh 
```

## Automatic deployment

You can automatically deploy the entire cluster (workers, GCS, global scheduler nodes) by following these instructions.

First, build everything:

```bash
make all
```

This will generate the Go and Python gRPC stub files, compile the Go server code, and build the Docker images we need to deploy the cluster.

Next, spin up the cluster:

```bash
docker-compose up
```

Now, your cluster should be running! Let's test out if a node can talk to another node. Note that these commands will now be run-specific, since Docker containers are ID'd  randomly. First, list the container ID's of our cluster with `docker ps`. We'll get something like:

```
CONTAINER ID   IMAGE                            COMMAND                  CREATED         STATUS         PORTS                      NAMES
c20387bbd534   ray-node                         "/bin/sh -c './go/bi…"   3 seconds ago   Up 2 seconds   0.0.0.0:50004->50004/tcp   babyray_worker3_1
dce69f377b01   ray-node                         "/bin/sh -c ./go/bin…"   3 seconds ago   Up 2 seconds   0.0.0.0:50001->50001/tcp   babyray_global_scheduler_1
e06583f90acd   ray-node                         "/bin/sh -c './go/bi…"   3 seconds ago   Up 2 seconds   0.0.0.0:50002->50002/tcp   babyray_worker1_1
1d2559b0bb90   ray-node                         "/bin/sh -c './go/bi…"   3 seconds ago   Up 2 seconds   0.0.0.0:50003->50003/tcp   babyray_worker2_1
c3781cf7bbce   ray-node                         "/bin/sh -c './go/bi…"   3 seconds ago   Up 2 seconds   0.0.0.0:50000->50000/tcp   babyray_gcs_1
```

Now, pick one of those container ID's---here we pick worker 1---and run an interactive shell on it:

```bash
docker exec -it e06583f90acd  /bin/bash
```

Then, run the command `ping node0` and you'll get something like:

```
docker exec -it e06583f90acd  /bin/bash
root@e06583f90acd:/app# ping node0
PING node0 (172.24.0.2) 56(84) bytes of data.
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=1 ttl=64 time=3.11 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=2 ttl=64 time=0.499 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=3 ttl=64 time=0.470 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=4 ttl=64 time=0.348 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=5 ttl=64 time=0.224 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=6 ttl=64 time=0.268 ms
64 bytes from babyray_gcs_1.babyray_mynetwork (172.24.0.2): icmp_seq=7 ttl=64 time=0.173 ms
```

Packets are flowing between worker 1 and the GCS!

### Shut down the cluster

Reset by doing ^C in the `docker-compose up` session and then running `docker-compose down` to make sure all the containers are killed and the network is deleted.

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

## Info for contributors (naming / standards)

### Usage Example

See [here in this file](https://github.com/rodrigo-castellon/babyray/blob/43848c2210b6b55912c873fdd4d749255190ab7f/go/cmd/worker/main.go#L58) for how to load DNS name + port number for a particular service in Go (in this case, GCS function table).

Note: that code is untested, so it may not work, but should give you something to work off of.

### Overview

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

All of this information is kept in [this config file](https://github.com/rodrigo-castellon/babyray/blob/main/config/app_config.yaml), so pull from it in your code.

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

# Developer notes

## Running go test individually
If you'd like to run `go test` individually, you should do the following:
```
cd babyray
pwd
```
On MacOS:
```
nano ~/.zshrc
export PROJECT_ROOT=/path/to/your/project
```
This ensures that your Go code has access to the `app_config.yaml` file.