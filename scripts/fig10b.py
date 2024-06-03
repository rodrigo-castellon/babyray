import sys
from babyray import (
    init,
    remote,
    get,
    shutdown,
)
import time
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

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
batch_size = 100_000

# Submit tasks without GCS flushing and measure memory usage
logger.info("Starting tasks submission without GCS flushing...")
for i in range(0, num_tasks, batch_size):
    futures = [f.remote() for _ in range(batch_size)]
    get(futures)
    
    # Measure memory usage
    usage = get_memory_usage()
    logger.info(f"Memory usage without flush at batch {i // batch_size}: {usage} MB")
    
    if usage > 8000:  # Stop if memory usage exceeds 8 GB
        break

# Reset babyray
shutdown()
init()

# Submit tasks with GCS flushing and measure memory usage
logger.info("Starting tasks submission with GCS flushing...")
for i in range(0, num_tasks, batch_size):
    futures = [f.remote() for _ in range(batch_size)]
    get(futures)
    
    # Measure memory usage
    usage = get_memory_usage()
    logger.info(f"Memory usage with flush at batch {i // batch_size}: {usage} MB")
    
    # Perform GCS flushing periodically
    if i % (10 * batch_size) == 0:
        flush_gcs()

# Shutdown babyray
shutdown()
