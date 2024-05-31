#!/bin/bash
# scripts/run_tests.sh

# Build and run the tests
# docker-compose build
docker-compose -f docker-compose.yml -f docker-compose.fig8a.yml up

# Wait for the services to be up and running
sleep 20

# Check the logs for the worker1 container to see if the tests passed
docker-compose logs worker1 | tee output.log

# Check if "All tests passed" is in the logs
if grep -q "1 passed" output.log; then
    echo "All tests passed!"
    docker-compose down
    exit 0
else
    echo "Some tests failed."
    docker-compose down
    exit 1
fi
