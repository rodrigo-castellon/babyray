import sys
from babyray import (
    init,
    remote,
    get,
    Future,
)
import time
from utils import log

log("FIG10b: HELLO WORLD I AM ANGRY!")

# Initialize babyray
init()

# Define a dummy remote function
@remote
def f():
    return 0

# Function to measure memory usage using /proc/meminfo
def get_memory_usage():
    with open('/proc/meminfo', 'r') as meminfo:
        lines = meminfo.readlines()
        meminfo_dict = {}
        for line in lines:
            parts = line.split(':')
            meminfo_dict[parts[0].strip()] = int(parts[1].strip().split()[0])  # Value is in KB

    total_memory = meminfo_dict['MemTotal'] / 1024  # Convert KB to MB
    free_memory = meminfo_dict['MemFree'] / 1024
    buffers = meminfo_dict['Buffers'] / 1024
    cached = meminfo_dict['Cached'] / 1024

    used_memory = total_memory - (free_memory + buffers + cached)
    return used_memory

# Function to perform GCS flushing
def flush_gcs():
    # Assuming there is a function in babyray for GCS flushing
    # Replace the following line with the actual flushing function if different
    babyray._private.internal_api.global_gc()

# Number of tasks to submit
num_tasks = 50_000_000

# Submit tasks without GCS flushing and measure memory usage
log("FIG10b: Starting tasks submission without GCS flushing...")
start_time = time.time()
elapsed_time = 0
total_duration = 2 * 60 * 60  # 2 hours in seconds

# Memory logging intervals
initial_interval = 1  # seconds
second_interval = 30  # seconds after initial period
initial_period = 300  # seconds
max_interval = 600  # Maximum interval of 10 minutes

current_interval = initial_interval
next_log_time = start_time + current_interval

i = 0

while elapsed_time < total_duration and i < num_tasks:
    # Submit a single task
    f.remote()
    i += 1

    # Measure memory usage
    current_time = time.time()
    elapsed_time = current_time - start_time

    usage = 0
    if current_time >= next_log_time:
        usage = get_memory_usage()
        log(f"FIG10b: Memory usage without flush after {i} tasks: {usage} MB")

        # Schedule next log time
        if elapsed_time < initial_period:
            current_interval = initial_interval
        else:
            current_interval = min(second_interval, max_interval)
        next_log_time = current_time + current_interval

    if usage > 8000:  # Stop if memory usage exceeds 8 GB
        break


# # Reset babyray
# init()

# # Submit tasks with GCS flushing and measure memory usage
# log("FIG10b: Starting tasks submission with GCS flushing...")
# start_time = time.time()
# elapsed_time = 0
# current_interval = initial_interval
# next_log_time = start_time + current_interval

# i = 0

# while elapsed_time < total_duration and i < num_tasks:
#     # Submit a single task
#     f.remote()
#     i += 1

#     # Measure memory usage
#     current_time = time.time()
#     elapsed_time = current_time - start_time

#     if current_time >= next_log_time:
#         usage = get_memory_usage()
#         log(f"Memory usage with flush after {i} tasks: {usage} MB")

#         # Schedule next log time
#         current_interval = min(current_interval * interval_increase_factor, max_interval)
#         next_log_time = current_time + current_interval

#     # Perform GCS flushing periodically
#     if i % (10 * int(current_interval)) == 0:
#         flush_gcs()

#     if elapsed_time >= total_duration:
#         break
