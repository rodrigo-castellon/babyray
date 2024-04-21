import grpc
from . import rayclient_pb2
from . import rayclient_pb2_grpc


def run():
    # Assuming the server is running on localhost at port 50051
    with grpc.insecure_channel("localhost:50051") as channel:
        # Create a stub (client)
        stub = rayclient_pb2_grpc.RayDriverStub(channel)

        # Create a Content message to send to the server
        content = rayclient_pb2.CommandRequest(command=b"Hello, Ray!")

        # Call the Put method
        output = stub.ExecuteCommand(content)
        print(f"output: {output}")

        # Call the Get method
        #retrieved_content = stub.Get(rayclient_pb2.ObjectId(id=object_id.id))
        #print(f"Retrieved content: {retrieved_content.data}")


if __name__ == "__main__":
    run()
