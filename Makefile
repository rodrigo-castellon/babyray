.PHONY: all go py clean build servers docker

PYTHON ?= python

all: go py build docker

go: proto
	@echo "Generating Go gRPC code..."
	protoc -I proto/ proto/rayclient.proto --go_out=go/pkg --go_opt=paths=source_relative --go-grpc_out=go/pkg --go-grpc_opt=paths=source_relative

py: proto
	@echo "Generating Python gRPC code..."
	$(PYTHON) -m grpc_tools.protoc -I proto/ --python_out=python/babyray --grpc_python_out=python/babyray proto/rayclient.proto
	@echo "Modifying import statements for relative imports..."
	# below line is now compatible with both MacOS (BSD) and GNU
	sed -i'' -e 's/import rayclient_pb2 as rayclient__pb2/from . import rayclient_pb2 as rayclient__pb2/' python/babyray/rayclient_pb2_grpc.py

build: servers

servers: gcsfunctable gcsobjtable globalscheduler localobjstore localscheduler worker

gcsfunctable:
	@echo "Building GCS Function Table Server..."
	cd go && go build -o bin/gcsfunctable cmd/gcsfunctable/*.go

gcsobjtable:
	@echo "Building GCS Object Table Server..."
	cd go && go build -o bin/gcsobjtable cmd/gcsobjtable/*.go

globalscheduler:
	@echo "Building Global Scheduler Server..."
	cd go && go build -o bin/globalscheduler cmd/globalscheduler/*.go

localobjstore:
	@echo "Building Local Object Store Server..."
	cd go && go build -o bin/localobjstore cmd/localobjstore/*.go

localscheduler:
	@echo "Building Local Scheduler Server..."
	cd go && go build -o bin/localscheduler cmd/localscheduler/*.go

worker:
	@echo "Building Worker Server..."
	cd go && go build -o bin/worker cmd/worker/*.go

docker:
	docker build --build-arg CONFIG=base -t ray-node:base .
	docker build --build-arg CONFIG=driver -t ray-node:driver .

clean:
	@echo "Cleaning up..."
	rm -f go/pkg/*.pb.go
	rm -f python/babyray/*_pb2.py
	rm -f python/babyray/*_pb2_grpc.py
	rm -f go/bin/*

