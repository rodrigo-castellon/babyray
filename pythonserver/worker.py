import grpc
from concurrent import futures
import time
import threading
from datetime import datetime
import cloudpickle as pickle
from babyray import init, get, remote, Future

from .constants import *
from . import rayclient_pb2
from . import rayclient_pb2_grpc

MAX_CONCURRENT_TASKS = 10
EMA_PARAM = 0.9

# Global variables
semaphore = threading.Semaphore(MAX_CONCURRENT_TASKS)
num_running_tasks = 0
num_queued_tasks = 0
average_running_time = 0.1
mu = threading.Lock()

MAX_MESSAGE_SIZE = 1024 * 1024 * 1024  # 1GB


def block_forever():
    while True:
        time.sleep(100)


gcs_func_gRPC = None
lobs_gRPC = None
server = None


def local_log(*s):
    # Get the current datetime
    current_datetime = datetime.now()

    # Format the datetime
    formatted_datetime = current_datetime.strftime("%Y/%m/%d %H:%M:%S.%f")[:-3]

    print(formatted_datetime, "[worker]", *s, flush=True)


def init_all_stubs():
    global gcs_func_gRPC, lobs_gRPC
    channel_options = [
        ("grpc.max_send_message_length", MAX_MESSAGE_SIZE),
        ("grpc.max_receive_message_length", MAX_MESSAGE_SIZE),
    ]

    gcs_func_channel = grpc.insecure_channel(
        f"node{GCS_NODE_ID}:{GCS_FUNCTION_TABLE_PORT}",
        options=channel_options,
    )
    gcs_func_gRPC = rayclient_pb2_grpc.GCSFuncStub(gcs_func_channel)

    lobs_channel = grpc.insecure_channel(
        f"localhost:{LOCAL_OBJECT_STORE_PORT}", options=channel_options
    )
    lobs_gRPC = rayclient_pb2_grpc.LocalObjStoreStub(
        lobs_channel,
    )


class WorkerServer(rayclient_pb2_grpc.WorkerServicer):
    def __init__(self):
        init()
        self.alive = True

    def Run(self, request, context):
        global num_running_tasks, num_queued_tasks, average_running_time

        if not self.alive:
            block_forever()

        local_log("in Run() rn")

        with mu:
            num_queued_tasks += 1

        with semaphore:
            with mu:
                num_queued_tasks -= 1
                num_running_tasks += 1

            try:
                start = time.time()

                func_response = gcs_func_gRPC.FetchFunc(
                    rayclient_pb2.FetchRequest(name=request.name)
                )
                func_obj = pickle.loads(func_response.serializedFunc)
                args_obj = pickle.loads(request.args)
                kwargs_obj = pickle.loads(request.kwargs)

                output = func_obj(*args_obj, **kwargs_obj)

                if not self.alive:
                    block_forever()

                output_pickled = pickle.dumps(output)
                local_log("Executed!")

                local_log("gonna store now")
                lobs_gRPC.Store(
                    rayclient_pb2.StoreRequest(
                        uid=request.uid, objectBytes=output_pickled
                    )
                )
                local_log("stored...")

                running_time = time.time() - start
                local_log(f"took this many seconds: {running_time:.3g}")

                with mu:
                    average_running_time = (
                        EMA_PARAM * average_running_time
                        + (1 - EMA_PARAM) * running_time
                    )
            finally:
                with mu:
                    num_running_tasks -= 1

        return rayclient_pb2.StatusResponse(success=True)

    def WorkerStatus(self, request, context):
        global num_running_tasks, num_queued_tasks, average_running_time
        if not self.alive:
            block_forever()

        with mu:
            return rayclient_pb2.WorkerStatusResponse(
                numRunningTasks=num_running_tasks,
                numQueuedTasks=num_queued_tasks,
                averageRunningTime=average_running_time,
            )

    def KillServer(self, request, context):
        # global task_threads
        local_log("GOT KILLED")
        self.alive = False
        return rayclient_pb2.StatusResponse(success=True)

    def ReviveServer(self, request, context):
        local_log("GOT REVIVED!")
        self.alive = True
        return rayclient_pb2.StatusResponse(success=True)


def serve():
    global server
    init_all_stubs()

    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10 * MAX_CONCURRENT_TASKS)
    )
    rayclient_pb2_grpc.add_WorkerServicer_to_server(WorkerServer(), server)

    server_address = "0.0.0.0:50002"
    server.add_insecure_port(server_address)
    local_log(f"Server listening on {server_address}")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
