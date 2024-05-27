import sys
import cloudpickle as pickle
import base64
from babyray import init

init()


def main():
    # Read binary data from stdin
    input_data = sys.stdin.buffer.read()

    # Split the input data into function, args, and kwargs based on newline separator
    # Assuming each segment ends with a newline, split by newline and remove the last empty part if it exists
    parts = input_data.split(b"\n")

    # Unpickle each part
    # The first part is the function, the second is args, the third is kwargs
    func = pickle.loads(base64.b64decode(parts[0]))
    args = pickle.loads(base64.b64decode(parts[1]))
    kwargs = pickle.loads(base64.b64decode(parts[2]))

    # Execute the function with the provided args and kwargs
    result = func(*args, **kwargs)

    # Print the result in base64 so that Go can take it
    print(base64.b64encode(pickle.dumps(result)).decode("ascii"), end="")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("exception was:", e)
