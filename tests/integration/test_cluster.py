# tests/integration/test_script.py

######## DISCLAIMER ########
# Some missing desired functionality (that is not included in these tests):
# - Remote functions cannot call remote functions defined below their own
#   definition.
# - Remote functions cannot call themselves.
# Solution to this is currently unclear.
######## DISCLAIMER ########

import pytest

import cloudpickle as pickle
from babyray import init, remote, get, Future


@pytest.fixture(scope="module", autouse=True)
def setup_ray():
    init()  # Initialize the Ray cluster


def test_simple():
    # simplest possible test

    @remote
    def f(x):
        return x * x

    a = f.remote(2)
    res = get(a)

    assert res == 4


def test_simple_list():
    @remote
    def f(x):
        return x * x

    futures = [f.remote(i) for i in range(4)]
    results = get(futures)

    assert results == [0, 1, 4, 9]


def test_nested_remote_functions():
    @remote
    def f(x):
        return x * x

    @remote
    def g(x):
        y = f.remote(x)
        return get(y) + 1

    a = g.remote(3)
    res = get(a)

    assert res == 10  # 3*3 + 1


def test_task_dependencies():
    @remote
    def f(x):
        return x * x

    @remote
    def g(x: Future, y: Future) -> int:
        return get(x) + get(y)

    a = f.remote(2)
    b = f.remote(3)
    c = g.remote(a, b)
    res = get(c)

    assert res == 13  # 2*2 + 3*3


def test_sequential_task_execution():
    @remote
    def f(x):
        return x + 1

    a = f.remote(1)
    b = f.remote(get(a))
    c = f.remote(get(b))
    res = get(c)

    assert res == 4  # 1 + 1 + 1 + 1


def test_complex_dag():
    @remote
    def f(x):
        return x + 1

    @remote
    def g(x, y):
        return x * y

    @remote
    def h(x):
        return x - 1

    a = f.remote(1)
    b = f.remote(2)
    c = g.remote(get(a), get(b))
    d = h.remote(get(c))
    e = f.remote(get(d))
    res = get(e)

    assert res == 6  # (((1 + 1) * (2 + 1)) - 1) + 1


def test_dynamic_task_creation():
    @remote
    def f(x):
        return x + 1

    @remote
    def g(x):
        # 3
        futures = [f.remote(x + i) for i in range(5)]
        return get(futures)

    a = f.remote(2)  # 3
    b = g.remote(get(a))
    res = get(b)

    assert res == [4, 5, 6, 7, 8]  # [2+1, 2+1+1, ..., 2+1+4]


def test_tree_structure():
    @remote
    def f(x):
        return x + 1

    @remote
    def combine(x, y):
        return x + y

    a = f.remote(1)  # 2
    b = f.remote(2)  # 3
    c = combine.remote(get(a), get(b))  # 5
    d = f.remote(get(c))  # 6
    e = f.remote(get(c))  # 6
    f = combine.remote(get(d), get(e))  # 12
    res = get(f)

    assert res == 12  # ((1+1) + (2+1)) + 1 + 1


import time


def test_varying_task_durations():
    @remote
    def f(x):
        time.sleep(x)
        return x

    durations = [0.1, 0.2, 0.3, 0.4, 0.5]
    futures = [f.remote(d) for d in durations]
    results = get(futures)

    assert results == durations


def test_high_concurrency():
    @remote
    def f(x):
        return x * x

    num_tasks = 1000
    futures = [f.remote(i) for i in range(num_tasks)]
    results = get(futures)

    assert results == [i * i for i in range(num_tasks)]


if __name__ == "__main__":
    pytest.main()
