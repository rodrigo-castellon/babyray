#!/bin/bash
# scripts/run_fig10b.sh

# REALLY ONLY USING THIS; EVERYTHING ELSE IS JUNK
docker-compose -f docker-compose.yml -f docker-compose.fig10b.yml up 


# ==================

# docker-compose logs worker1 > out.txt




# Wait for the services to be up and running
# sleep 

# Get current timestamp
# timestamp=$(date +"%Y%m%d_%H%M%S")

# # Define the log filename with the timestamp suffix
# log_filename="output_${timestamp}.log"

# Capture and filter the logs, then save to the timestamped file
# docker-compose logs worker1 | grep FIG10b: | tee "$log_filename"
# docker-compose logs worker1 | tee "$log_filename"
# docker-compose logs worker1 | grep FIG10b:
#echo yolo
# docker-compose logs worker1 > out.txt

# # Check if "1 passed" is in the logs
# # if grep -q "1 passed" "$log_filename"; then
# if grep -q "1 passed" "output_hello.log"; then
#     echo "All tests passed!"
#     docker-compose down
#     exit 0
# else
#     echo "Some tests failed."
#     docker-compose down
#     exit 1
# fi
