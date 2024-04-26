from .client import run
from .utils import *

from . import rayclient_pb2
from . import rayclient_pb2_grpc

local_scheduler_gRPC = None


def init_all_stubs():
    channel = grpc.insecure_channel("localhost:" + port)
    local_scheduler_gRPC = rayclient_pb2_grpc.LocalSchedulerStub(channel)


class LocalSchedulergRPC:
    def __init__(self):
        pass

    def schedule(self):
        pass


class GCSFuncgRPC:
    def __init__(self):
        pass

    def schedule(self, name, args, kwargs):
        pass


class Future:
    def __init__(self, val=None):
        self.val = val

    def get(self):
        return self.val


class RemoteFunction:
    def __init__(self, func):
        self.func = func
        self.name = self.func.__name__ + str(get_uuid())

        # create the stub

    def remote(self, *args, **kwargs) -> Future:
        # do gRPC here

        # returns a future
        out = local_scheduler_gRPC.schedule(self.name, args, kwargs)
        return Future(out)

    def register(self):
        # do gRPC here
        serialized = pickle.dumps(self.func)

        gcs_func_gRPC.register(self.name, serialized)


def init():
    print("Initializing Baby Ray")
    # we need to connect to:
    # gcs function table
    # gcs object table
    # global scheduler
    global local_scheduler_gRPC
    local_scheduler_gRPC = init_all_stubs()


# You can define decorators and functions for remote execution here.
# This is just a basic template.


def remote(func):
    rem = RemoteFunction(func)
    return rem


def get(future):
    # traverse
    vals = [f.get() for f in future]
    return vals


# Example function to simulate Ray's behavior
@remote
def f(x):
    return x * x
