.PHONY: all go py clean

all: go py

go: proto
	@echo "Generating Go gRPC code..."
	protoc -I proto/ proto/rayclient.proto --go_out=go/pkg --go_opt=paths=source_relative --go-grpc_out=go/pkg --go-grpc_opt=paths=source_relative

py: proto
	@echo "Generating Python gRPC code..."
	python -m grpc_tools.protoc -I proto/ --python_out=python/babyray --grpc_python_out=python/babyray proto/rayclient.proto

clean:
	@echo "Cleaning up..."
	rm -f go/pkg/*.go
	rm -f python/babyray/*_pb2.py
	rm -f python/babyray/*_pb2_grpc.py
