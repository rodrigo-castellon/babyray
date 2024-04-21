.PHONY: all go py clean build servers

all: go py build

go: proto
	@echo "Generating Go gRPC code..."
	protoc -I proto/ proto/rayclient.proto --go_out=go/pkg --go_opt=paths=source_relative --go-grpc_out=go/pkg --go-grpc_opt=paths=source_relative

py: proto
	@echo "Generating Python gRPC code..."
	python -m grpc_tools.protoc -I proto/ --python_out=python/babyray --grpc_python_out=python/babyray proto/rayclient.proto

build: servers

servers: driver gcs globalscheduler localscheduler worker

driver:
	@echo "Building Driver Server..."
	cd go && go build -o bin/driver cmd/driver/main.go

gcs:
	@echo "Building GCS Server..."
	cd go && go build -o bin/gcs cmd/gcs/main.go

globalscheduler:
	@echo "Building Global Scheduler Server..."
	cd go && go build -o bin/globalscheduler cmd/globalscheduler/main.go

localscheduler:
	@echo "Building Local Scheduler Server..."
	cd go && go build -o bin/localscheduler cmd/localscheduler/main.go

worker:
	@echo "Building Worker Server..."
	cd go && go build -o bin/worker cmd/worker/main.go

clean:
	@echo "Cleaning up..."
	rm -f go/pkg/*.pb.go
	rm -f python/babyray/*_pb2.py
	rm -f python/babyray/*_pb2_grpc.py
	rm -f go/bin/*

