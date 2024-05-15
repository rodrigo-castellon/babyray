#!/bin/bash

# start an interactive /bin/bash session on the first container returned by "docker ps" that 
# uses the "ray-node:driver" image
docker exec -it $(docker ps | grep 'ray-node:driver' | awk '{print $1}' | head -1) /bin/bash
