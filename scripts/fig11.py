import sys
from babyray import init, remote, get, Future, kill_node
import random
import time

from utils import *


init()


# client-side code


# ask for a node that is not ourself
@remote
def f():
    time.sleep(5)
    return 0


f.set_node(3)

fut = f.remote()

time.sleep(2)

kill_node(3)

# here: watch the docker compose logs to see if things go the way we expect

out = get(fut)
log("out", out)
