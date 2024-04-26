# some constants, like the node ID's for the GCS, scheduler, and worker nodes

NUM_WORKER_NODES = 5  # includes ourself
CLUSTER_SIZE = NUM_WORKER_NODES + 2  # + GCS + global scheduler

# we reserve the following node id's:
    # 0: GCS
    # 1: global scheduler
    # 2: ourself (driver/worker node)
# the rest of the node id's >2 are worker nodes


GCS_NODE_ID = 0
GLOBAL_SCHEDULER_NODE_ID = 1
OURSELF_NODE_ID = 2

# containers are named as follows
# port numbers are named as follows:
# str(5000) + str(NODE_ID)
# so, 50000, 5


# we use DNS to assign each container a separate name, named
# "nodeX" where "X" is the integer node ID

# furthermore, we assign ports to correspond to services:
# 50000: local object store
# 50001: local scheduler
# 60000: worker 0
# 60001: worker 1
# 60002: worker 2
# and so on

