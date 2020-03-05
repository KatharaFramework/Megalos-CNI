import os
from kubernetes import client, config, watch


VTYSH_COMMAND_TEMPLATE = [
                            "vtysh",
                            "-c \"configure terminal\"",
                            "-c \"router bgp 65000\"",
                            "-c \"#neighbor#\"",
                            "-c \"exit\"",
                            "-c \"exit\""
                        ]
NEIGHBOR_STRING_TEMPLATE = "neighbor %s peer-group fabric"

KUBELET_CONFIG_PATH = "/host/etc/kubernetes/kubelet.conf"
FRR_CONFIG_DIR = "/etc/frr"
IS_MASTER = (os.environ.get("IS_MASTER") == "true")
NODE_IP = os.environ.get("NODE_IP")


def init_frr():
    # Checks which BGP configuration this node should use
    config_to_read = "master" if IS_MASTER else "worker"

    # Opens the stub and reads it
    with open("%s/bgpd_%s.stub" % (FRR_CONFIG_DIR, config_to_read), "r") as bgpd_config_stub_file:
        bgpd_config_stub = bgpd_config_stub_file.read()

    # Apply node IPs
    config_string = (bgpd_config_stub % (NODE_IP, NODE_IP)) if IS_MASTER else (bgpd_config_stub % NODE_IP)

    # Write the desired BGP configuration in FRR configuration file
    with open("%s/frr.conf" % FRR_CONFIG_DIR, "w") as frr_config:
        frr_config.write(config_string)

    # Start FRR
    os.system("/etc/init.d/frr start")

    # Remove stubs after FRR is started
    os.remove("%s/bgpd_master.stub" % FRR_CONFIG_DIR)
    os.remove("%s/bgpd_worker.stub" % FRR_CONFIG_DIR)


def start_k8s_watch(v1_client, watch_client):
    for event in watch_client.stream(v1_client.list_node, timeout_seconds=60):
        event_type = event['type']
        event_object = event['object']

        # Check if event node is master or not
        node_is_master = "node-role.kubernetes.io/master" in event_object.metadata.labels

        # Do stuff only if:
        # 1- The current node is a master and the event node is not a master
        # 2- The current node is not a master and the event node is a master
        if (IS_MASTER and not node_is_master) or (not IS_MASTER and node_is_master):
            # Get the node status
            node_status = event_object.status
            # Get the IP address from the status
            ip_address = [x.address for x in node_status.addresses if x.type == "InternalIP"].pop()

            if event_type == "ADDED":
                bgp_neighbor("add", ip_address)            # When a node is added, add it as BGP neighbor
            elif event_type == "DELETED":
                bgp_neighbor("del", ip_address)            # When a node is removed, delete it as BGP neighbor
            elif event_type == "MODIFIED":                 # When a node is modified, check if it's ready or not
                status = [x.status for x in node_status.conditions if x.type == "Ready"].pop()

                if status != "True":                # If not, delete it as BGP neighbor
                    bgp_neighbor("del", ip_address)
                else:                               # If yes, add it as BGP neighbor
                    bgp_neighbor("add", ip_address)


def bgp_neighbor(event, ip_address):
    # Adds "no" before the neighbor command if a peer should be removed
    prefix = "" if event == "add" else "no "
    neighbor_string = prefix + (NEIGHBOR_STRING_TEMPLATE % ip_address)

    # Replaces the neighbor command in the vtysh template command
    command_to_execute = [
        x.replace("#neighbor#", neighbor_string) if "#neighbor#" in x else x for x in VTYSH_COMMAND_TEMPLATE
    ]

    # Exec the vtysh command
    os.system(" ".join(command_to_execute))


if __name__ == '__main__':
    # Init FRR BGP configuration based on current node
    init_frr()

    # Starts Kubernetes Node Watcher on this node
    config.load_incluster_config()

    v1 = client.CoreV1Api()
    w = watch.Watch()

    while True:
        start_k8s_watch(v1, w)
