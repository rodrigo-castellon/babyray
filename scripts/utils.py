from datetime import datetime


def log(*s):
    # Get the current datetime
    current_datetime = datetime.now()

    # Format the datetime
    formatted_datetime = current_datetime.strftime("%Y/%m/%d %H:%M:%S.%f")[:-3]

    print(formatted_datetime, *s, flush=True)
