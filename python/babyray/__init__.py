# generic libraries
import dill as pickle
import grpc

from dataclasses import dataclass

# local imports
from .client import run
from .utils import *
from .constants import *

from . import rayclient_pb2
from . import rayclient_pb2_grpc

local_scheduler_gRPC = None
gcs_func_gRPC = None
local_object_store_gRPC = None


def init_all_stubs():
    # so boilerplate-y
    local_scheduler_channel = grpc.insecure_channel(
        f"localhost:{str(LOCAL_SCHEDULER_PORT)}"
    )
    local_scheduler_gRPC = rayclient_pb2_grpc.LocalSchedulerStub(
        local_scheduler_channel
    )

    gcs_func_channel = grpc.insecure_channel(
        f"node{GCS_NODE_ID}:{GCS_FUNCTION_TABLE_PORT}"
    )
    gcs_func_gRPC = rayclient_pb2_grpc.GCSFuncStub(gcs_func_channel)

    local_object_store_channel = grpc.insecure_channel(
        f"localhost:{str(LOCAL_OBJECT_STORE_PORT)}"
    )
    local_object_store_gRPC = rayclient_pb2_grpc.LocalObjStoreStub(
        local_object_store_channel
    )

    return local_scheduler_gRPC, gcs_func_gRPC, local_object_store_gRPC


@dataclass
class Future:
    uid: int

    def get(self):
        # make a request to local object store
        out = local_object_store_gRPC.Get(rayclient_pb2.GetRequest(uid=self.uid))
        return out


class RemoteFunction:
    def __init__(self, func):
        self.func = func
        self.name = None

    def remote(self, *args, **kwargs):
        # do gRPC here

        if self.name is not None:
            uid = local_scheduler_gRPC.Schedule(
                rayclient_pb2.ScheduleRequest(
                    name=self.name, args=pickle.dumps(args), kwargs=pickle.dumps(kwargs)
                )
            )
            return Future(uid)

    def register(self):
        # get our unique name from GCS
        self.name = gcs_func_gRPC.RegisterFunc(
            rayclient_pb2.RegisterRequest(
                serializedFunc=pickle.dumps(self.func)
            )
        )


def init():
    print("Initializing Baby Ray")
    # we need to connect to:
    # gcs function table
    # gcs object table
    # global scheduler
    global local_scheduler_gRPC, gcs_func_gRPC, local_object_store_gRPC
    local_scheduler_gRPC, gcs_func_gRPC, local_object_store_gRPC = init_all_stubs()


# You can define decorators and functions for remote execution here.
# This is just a basic template.


def remote(func):
    rem = RemoteFunction(func)
    rem.register()
    return rem


def get(futures):
    # recursive function
    if isinstance(futures, list):
        return [get(future) for future in futures]
    elif isinstance(futures, dict):
        return {k: get(future) for k, future in futures.items()}
    else:
        return futures.get() if hasattr(futures, "get") else futures


def demo():
    # Example function to simulate Ray's behavior
    @remote
    def f(x):
        return x * x

    print("running f(3)")
    future = f.remote(3)
    print("got future:", future)
    out = future.get()
    print("output is:", out)
