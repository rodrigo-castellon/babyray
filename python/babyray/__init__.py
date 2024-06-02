# generic libraries
import cloudpickle as pickle
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

MAX_MESSAGE_SIZE = 1024 * 1024 * 1024  # 1GB


def init_all_stubs():
    channel_options = [
        ("grpc.max_send_message_length", MAX_MESSAGE_SIZE),
        ("grpc.max_receive_message_length", MAX_MESSAGE_SIZE),
    ]

    # so boilerplate-y
    local_scheduler_channel = grpc.insecure_channel(
        f"localhost:{str(LOCAL_SCHEDULER_PORT)}",
        options=channel_options,
    )
    local_scheduler_gRPC = rayclient_pb2_grpc.LocalSchedulerStub(
        local_scheduler_channel
    )

    gcs_func_channel = grpc.insecure_channel(
        f"node{GCS_NODE_ID}:{GCS_FUNCTION_TABLE_PORT}",
        options=channel_options,
    )
    gcs_func_gRPC = rayclient_pb2_grpc.GCSFuncStub(gcs_func_channel)

    local_object_store_channel = grpc.insecure_channel(
        f"localhost:{str(LOCAL_OBJECT_STORE_PORT)}",
        options=channel_options,
    )
    local_object_store_gRPC = rayclient_pb2_grpc.LocalObjStoreStub(
        local_object_store_channel
    )

    return local_scheduler_gRPC, gcs_func_gRPC, local_object_store_gRPC


@dataclass
class Future:
    uid: int

    def get(self, copy=True, cache=True):
        # copy: whether we should actually copy the data over, or whether
        # we should just block until the task is complete
        # cache: whether we should cache it locally in our LOBS

        # make a request to local object store
        if copy:
            return pickle.loads(
                local_object_store_gRPC.Get(
                    rayclient_pb2.GetRequest(uid=self.uid, copy=True, cache=cache)
                ).objectBytes
            )
        else:
            local_object_store_gRPC.Get(
                rayclient_pb2.GetRequest(uid=self.uid, copy=False, cache=cache)
            )
            return True


class RemoteFunction:
    def __init__(self, func):
        self.func = func
        self.name = None
        self.node_id = None
        self.locality_flag = True

    """These three methods exist purely for experimental reasons."""

    def set_locality_flag(self, locality_flag):
        self.locality_flag = locality_flag

    def set_node(self, node_id):
        # set the node ID to run this remote function on
        self.node_id = node_id

    def unset_node(self):
        self.node_id = None

    def remote(self, *args, **kwargs):
        # do gRPC here
        arg_uids = []
        for arg in args:
            if type(arg) is Future:
                arg_uids.append(arg.uid)

        if self.name is not None:
            uid = local_scheduler_gRPC.Schedule(
                rayclient_pb2.ScheduleRequest(
                    name=self.name,
                    args=pickle.dumps(args),
                    kwargs=pickle.dumps(kwargs),
                    uids=arg_uids,
                    nodeId=self.node_id,
                    localityFlag=self.locality_flag,
                )
            ).uid
            return Future(uid)

    def register(self):
        # get our unique name from GCS
        self.name = gcs_func_gRPC.RegisterFunc(
            rayclient_pb2.RegisterRequest(serializedFunc=pickle.dumps(self.func))
        ).name


def init():
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


def get(futures, copy=True, cache=True):
    # recursive function
    if isinstance(futures, list):
        return [get(future, copy=copy, cache=cache) for future in futures]
    elif isinstance(futures, dict):
        return {k: get(future, copy=copy, cache=cache) for k, future in futures.items()}
    else:
        return (
            futures.get(copy=copy, cache=cache) if hasattr(futures, "get") else futures
        )


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
