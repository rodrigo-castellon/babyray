# tests/integration/test_script.py
import pytest

# import dill as pickle
import cloudpickle as pickle
from babyray import init, remote, get


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


def test_high_concurrency():
    @remote
    def f(x):
        return x * x

    num_tasks = 1000
    futures = [
        f.remote(i) for i in range(num_tasks)
    ]
    results = get(futures)

    assert results == [
        i * i for i in range(num_tasks)
    ]


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
    def g(x, y):
        return x + y

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


if __name__ == "__main__":
    pytest.main()
