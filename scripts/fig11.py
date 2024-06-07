import sys
from babyray import init, remote, get, Future, kill_node
import random
import time

from utils import *

# (pseudo)code for figure 11a

init()

DELAY = 0.1  # 100ms, per the original paper
BEGIN_INTERVAL = 50  # 50s, per the original paper
SECOND_INTERVAL = 40  # ~40s, per the original paper
THIRD_INTERVAL = 120  # ~120s, per the original paper


@remote
def firstfunc():
    # time.sleep(DELAY)
    return 0


def create_chain(n):

    funcs = [firstfunc]
    for i in range(80):

        @remote
        def func():
            time.sleep(DELAY)
            out = get(funcs[-1].remote())
            print("out", out)
            return out + 1

        funcs.append(func)

    return funcs


funcs = create_chain(30)
funcs2 = create_chain(30)

# probably need to manually define all of these because Python issues


out = funcs[-1].remote()
out2 = funcs2[-1].remote()

output = get((out, out2))

print("the output was", output)
# out = func1.remote()
# keep eye on logs from here on out
# time.sleep(BEGIN_INTERVAL)
# kill_node(3)
# time.sleep(SECOND_INTERVAL)
# kill_node(4)
# time.sleep(THIRD_INTERVAL)
# revive_node(3)
# revive_node(4)
