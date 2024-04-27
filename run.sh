# run a single ray node docker container
docker build -t worker-node . && docker run --rm -it worker-node
