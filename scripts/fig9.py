# pseudocode for Figure 9

# server-side modifications needed:
# - local scheduler and global scheduler should work at all
# - local scheduler: should be able to shuttle not_client=True to global
# - global scheduler: should be able to pick something that is not_client. doesn't matter how it's actually picked.
# also, it doesn't seem like we actually need multiple VMs for this

import sys
from babyray import (
    init,
    remote,
    get,
    Future,
)
import random
import time

from utils import *

init()

# client-side code

MAX_TIME = 1.0  # just time 1 second at a time


# ask for a node that is not ourself
@remote
def f(size):
    return bytearray(size)


def get_iops(size, max_time):
    counter = 0
    start = time.time()
    while time.time() - start < max_time:
        get(f.remote(size), copy=False)
        counter += 1

    return counter


NUM_TRIALS = 3
sizes = [1, 10, 100, 1_000, 10_000, 100_000, 1_000_000]  # in KBs
sizes = [size * 1000 for size in sizes]
max_times = [1.0] * 4 + [5.0, 50, 5 * 60]
f.set_node(2)
for size in sizes:
    for i in range(NUM_TRIALS):
        # log("creating the object")
        # uid = f.remote(size)
        # get(uid, copy=False)  # block until it completes
        # log("object created")
        start = time.time()
        get(f.remote(size), copy=False)
        elapsed = time.time() - start
        iops = 1 / elapsed
        thpt = iops * size

        log(f"RES:{size},{iops},{thpt}")

log("DONE!")
