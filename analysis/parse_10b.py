import unittest
from io import StringIO

def parse_log(log_data):
    log_data = log_data.splitlines()

    output = []
    for line in log_data:
        try:
            timestamp, memory_usage = line.split('FIG10b: Memory usage without flush after')[0].rstrip(), line.split('MB')[-1].strip()
            output.append([timestamp, memory_usage])
        except ValueError:
            pass

    return output

class TestParseLog(unittest.TestCase):
    def test_parse_log(self):
        # Test case 1: Valid log data
        valid_log_data = """2024/06/04 01:44:07.510 FIG10b: Memory usage without flush after 123 tasks: 970.90234375 MB
2024/06/04 01:44:08.529 FIG10b: Memory usage without flush after 245 tasks: 977.0 MB
2024/06/04 01:44:09.530 FIG10b: Memory usage without flush after 373 tasks: 986.859375 MB
2024/06/04 01:44:10.533 FIG10b: Memory usage without flush after 500 tasks: 987.625 MB
"""
        expected_output = [
            ['2024/06/04 01:44:07.510', '970.90234375'],
            ['2024/06/04 01:44:08.529', '977.0'],
            ['2024/06/04 01:44:09.530', '986.859375'],
            ['2024/06/04 01:44:10.533', '987.625']
        ]
        self.assertListEqual(parse_log(valid_log_data), expected_output)

        # Test case 2: Log data with malformed lines
        malformed_log_data = """2024/06/04 01:44:07.510 FIG10b: Memory usage without flush after 123 tasks: 970.90234375 MB
This is a malformed line
2024/06/04 01:44:09.530 FIG10b: Memory usage without flush after 373 tasks: 986.859375 MB
Another malformed line
2024/06/04 01:44:10.533 FIG10b: Memory usage without flush after 500 tasks: 987.625 MB
"""
        expected_output = [
            ['2024/06/04 01:44:07.510', '970.90234375'],
            ['2024/06/04 01:44:09.530', '986.859375'],
            ['2024/06/04 01:44:10.533', '987.625']
        ]
        self.assertListEqual(parse_log(malformed_log_data), expected_output)

if __name__ == '__main__':
    unittest.main()