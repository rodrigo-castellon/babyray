import sys
from babyray import (
    init,
    remote,
    get,
    Future,
)
from utils import *


init()

# worker 1:
# - create object #1
# - schedule 10 sleeper tasks
# - while True: time doing get(task(object #1)) or get(task(object #2))
# worker 2:
# - create object #2

import time
import random

# 100KB to start
# BYTEARR_SIZE = 100_000
BYTEARR_SIZE = 200_000_000  # 100_000_000  # 100 MB


# just create a local bytearray of size n
@remote
def create_bytearr(n):
    return bytearray(n)


@remote
def dummy_func(fut):
    bytearr = get(fut, pickle_load=False)

    return 10


@remote
def sleeper_func():
    time.sleep(999999)
    return ""


# sleeper functions are to ensure that any task we try to run will go to the
# global scheduler first (instead of routed only locally)
log("running sleeper functions")
for i in range(11):
    sleeper_func.remote()
    log("wtf", i)

log("done????")
time.sleep(1)
log("done with deploying sleepers")

SIZES = [100_000, 1_000_000, 10_000_000, 100_000_000]
# SIZES = [100_000_000]
# SIZES = [10_000] * 4
NUM_TRIALS = 10

for size in SIZES:
    log("#" * 100)
    log(f"DOING SIZE = {size}")
    log("#" * 100)

    for i in range(NUM_TRIALS):
        # create an object either on node 3 or node 4
        # if random.random() < 0.5:
        create_bytearr.set_node(3)
        # else:
        #     create_bytearr.set_node(4)

        log("now actually creating the object...")
        obj = create_bytearr.remote(size)
        log("its created. now lets get it.")
        get(obj, copy=False)  # block on this task
        # time.sleep(2)

        log("#" * 100)
        log("DOING LOCALITY-AWARE")
        log("#" * 100)
        # dummy_func.set_node(3)
        dummy_func.set_locality_flag(True)
        start = time.time()
        out = get(dummy_func.remote(obj))
        elapsed = time.time() - start

        log(f"SIZE={size} | SAME NODE | elapsed time:", elapsed)

        log("#" * 100)
        log("DOING LOCALITY UNAWARE")
        log("#" * 100)

        dummy_func.set_locality_flag(False)
        start = time.time()
        out = get(dummy_func.remote(obj))
        elapsed = time.time() - start

        log(f"SIZE={size} | DIFFERENT NODE | elapsed time:", elapsed)

# creating a byte array of size BYTEARR_SIZE on node 3
# create_bytearr.set_node(3)
# obj1 = create_bytearr.remote(BYTEARR_SIZE)
# log("obj1", obj1)
# time.sleep(1)

# # creating a byte array of size BYTEARR_SIZE on node 4
# log("creating obj2 now...")
# create_bytearr.set_node(4)
# obj2 = create_bytearr.remote(BYTEARR_SIZE)
# log("obj2", obj2)
# time.sleep(1)

# queue a task that relies on this object. will be run randomly if
# not locality-aware. will be run on the right node if locality-aware.

# for i in range(10):
#     start = time.time()
#     fut = dummy_func.remote(obj1)
#     log(get(fut))
#     elapsed = time.time() - start
#     log("elapsed time", elapsed)

# for i in range(1):
#     start = time.time()
#     # if random.random() < 0.5:
#     # obj_to_feed = obj1 if random.random() < 0.5 else obj2
#     fut = dummy_func.remote(obj1)
#     log("fut?", fut)

#     get(fut)

#     elapsed = time.time() - start

#     log("time:", elapsed)
# do the timer evaluation
# for i in range(NUM_TRIALS):
#     start = time.time()

#     if random.random() < 0.5:
#         get(dummy_func.remote(obj1))
#     else:
#         get(dummy_func.remote(obj2))

#     elapsed = time.time() - start
#     times.append(elapsed)

# print("total time:", sum(times) / len(times))
# with open("log.txt", "w") as f:
#     for time in times:
#         f.write(str(time) + ",")
