import re
import csv
from datetime import datetime

def parse_log_line(line):
    pattern = r'^worker\d+\-\d+\s+\|\s+(\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\.\d{3})\s+FIG10b:\s+Memory\s+usage\s+without\s+flush\s+after\s+(\d+)\s+tasks:\s+(\d+\.\d+)\s+MB'
    match = re.match(pattern, line)
    if match:
        timestamp = datetime.strptime(match.group(1), '%Y/%m/%d %H:%M:%S.%f')
        tasks = int(match.group(2))
        memory_usage = float(match.group(3))
        return (timestamp, memory_usage)
    else:
        return None

def parse_logs(log_data):
    result = []
    for line in log_data.split('\n'):
        parsed = parse_log_line(line)
        if parsed:
            result.append(parsed)
    return result

def write_to_csv(data, filename):
    with open(filename, 'w', newline='') as csvfile:
        writer = csv.writer(csvfile)
        writer.writerow(['Timestamp', 'Memory Usage (MB)'])
        for timestamp, memory_usage in data:
            writer.writerow([timestamp, memory_usage])

# Unit test
def test_parse_log_line():
    test_line = 'worker1-1  | 2024/06/04 05:02:21.134 FIG10b: Memory usage without flush after 1459 tasks: 1060.06640625 MB'
    expected_output = (datetime(2024, 6, 4, 5, 2, 21, 134000), 1060.06640625)
    assert parse_log_line(test_line) == expected_output

# Example usage
if __name__ == '__main__':
    test_parse_log_line()  # Run the unit test
    print("unit test worked")

    with open('output_10b_run2_300s_flush_2b_memory_limit.txt', 'r') as f:
        log_data = f.read()

    parsed_logs = parse_logs(log_data)
    write_to_csv(parsed_logs, 'output_10b_run2_300s_flush_2b_memory_limit_cleaned.csv')

    # =================================

    with open('output_10b_run3_40s_flush_2gb_memory_limit.txt', 'r') as f:
        log_data = f.read()

    parsed_logs = parse_logs(log_data)
    write_to_csv(parsed_logs, 'output_10b_run3_40s_flush_2gb_memory_limit_cleaned.csv')