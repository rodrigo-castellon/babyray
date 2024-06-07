from babyray import init, remote, get, kill_node
import time

max_concurrent_tasks = 10

init()


@remote
def short_sleeper():
    time.sleep(10)
    return 0


@remote
def generate_big_object(n):
    return bytearray(n)


@remote
def use_big_object(x):
    return get(x)


@remote
def task_A():
    time.sleep(10)
    return 0


task_A.set_node(3)


@remote
def task_B():
    return get(task_A.remote()) + 1


def bottom_up():
    # bottom up scheduling

    sleepers = []
    # submit sleeper tasks
    _ = input("submit sleeper tasks [ENTER]:")
    for _ in range(max_concurrent_tasks):
        sleepers.append(short_sleeper.remote())

    # submit sleeper tasks to other nodes
    for _ in range(5):
        _ = input("submit an extra sleeper task [ENTER]:")
        sleepers.append(short_sleeper.remote())

    get(sleepers)  # wait on all of these to complete


def task_locality():
    # task locality
    _ = input("Generate a big object on Worker 2 [ENTER]:")

    generate_big_object.set_node(3)

    x = generate_big_object.remote(2048)

    _ = input("Submit sleeper tasks to Worker 1 [ENTER]:")

    sleepers = []
    for _ in range(max_concurrent_tasks):
        sleepers.append(short_sleeper.remote())

    _ = input("Submit task that uses the big object [ENTER]")
    use_big_object.remote(x)

    get(sleepers)


def fault_tolerance():
    # Fault Tolerance
    # schedule task A on worker 2 = node 3
    # _ = input("Scheduling task A on worker 2 [ENTER]:")
    # task_A.set_node(3)
    # out_A = task_A.remote()

    # kill node 1
    # _ = input("Kill Worker 2 [ENTER]:")
    # kill_node(3)

    _ = input("Run task B, which depends on task A [ENTER]:")
    out_B = task_B.remote()

    _ = input("Kill task A's node [ENTER]:")
    kill_node(3)

    print("output: ", get(out_B))


def scripting():
    # bottom_up()
    # task_locality()
    fault_tolerance()


if __name__ == "__main__":
    scripting()
