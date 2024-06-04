import matplotlib.pyplot as plt
import csv
from datetime import datetime

# Read data from the CSV file
x1 = []  # Elapsed time (seconds)
y1 = []  # GCS Used Memory (MB)

with open('output_10b_run1_cleaned.csv', 'r') as csvfile:
    plots = csv.reader(csvfile)
    header = next(plots)  # Skip the header row
    timestamp_baseline = None
    memory_baseline = None

    for row in plots:
        timestamp_str, memory = row
        timestamp = datetime.strptime(timestamp_str, '%Y-%m-%d %H:%M:%S.%f')

        if timestamp_baseline is None:
            timestamp_baseline = timestamp
            memory_baseline = float(memory)

        elapsed_time = (timestamp - timestamp_baseline).total_seconds()
        memory_used = float(memory) - memory_baseline

        x1.append(elapsed_time)
        y1.append(memory_used)

# =====
# Read data from the CSV file
x2 = []  # Elapsed time (seconds)
y2 = []  # GCS Used Memory (MB)

with open('output_10b_run3_40s_flush_2gb_memory_limit_cleaned.csv', 'r') as csvfile:
    plots = csv.reader(csvfile)
    header = next(plots)  # Skip the header row
    timestamp_baseline = None
    memory_baseline = None

    for row in plots:
        timestamp_str, memory = row
        timestamp = datetime.strptime(timestamp_str, '%Y-%m-%d %H:%M:%S.%f')

        if timestamp_baseline is None:
            timestamp_baseline = timestamp
            memory_baseline = float(memory)

        elapsed_time = (timestamp - timestamp_baseline).total_seconds()
        memory_used = float(memory) - memory_baseline

        x2.append(elapsed_time)
        y2.append(memory_used)

# =====

# Read data from the CSV file
x3 = []  # Elapsed time (seconds)
y3 = []  # GCS Used Memory (MB)

with open('output_10b_run2_300s_flush_2b_memory_limit_cleaned.csv', 'r') as csvfile:
    plots = csv.reader(csvfile)
    header = next(plots)  # Skip the header row
    timestamp_baseline = None
    memory_baseline = None

    for row in plots:
        timestamp_str, memory = row
        timestamp = datetime.strptime(timestamp_str, '%Y-%m-%d %H:%M:%S.%f')

        if timestamp_baseline is None:
            timestamp_baseline = timestamp
            memory_baseline = float(memory)

        elapsed_time = (timestamp - timestamp_baseline).total_seconds()
        memory_used = float(memory) - memory_baseline

        x3.append(elapsed_time)
        y3.append(memory_used)

# =====

def attempt1():
    # Create the plot
    plt.plot(x1, y1, label="Baby Ray, no GCS flush", linestyle='--', color='blue')
    plt.plot(x2, y2, label="Baby Ray, GCS flush per 40s", color='orange')

    # Customize the plot
    plt.xlabel("Elapsed time (seconds)")
    plt.ylabel("GCS Used Memory (MB)")
    plt.title("Memory Usage Over Time")
    plt.legend(loc="lower right")

    # Show the plot
    plt.grid(True)
    plt.show()

def attempt2():
    # Create the plot
    plt.plot(x1, y1, label="Baby Ray, no GCS flush", linestyle='--', color='blue')
    plt.plot(x3, y3, label="Baby Ray, GCS flush per 300s", color='orange')

    # Customize the plot
    plt.xlabel("Elapsed time (seconds)")
    plt.ylabel("GCS Used Memory (MB)")
    plt.title("50 million no-op tasks")
    plt.legend(loc="lower right")

    # Show the plot
    plt.grid(True)
    plt.show()

if __name__ == '__main__':
    attempt2()