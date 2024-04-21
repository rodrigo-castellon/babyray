from .client import run
from .utils import some_utility_function

class Future:
    def __init__(self, val=None):
        self.val = val

    def get(self):
        return self.val

class RemoteFunction:
    def __init__(self, func):
        self.func = func
    def remote(self, *args):
        return Future(self.func(*args))

def init():
    print("Initializing Baby Ray")
    # any other initialization code

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

