import logging
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

FRR_CONFIG_DIR = "/etc/frr"
IS_MASTER = (os.environ.get("IS_MASTER") == "true")

service_ips = []


def init_frr():
    # Checks which BGP configuration this node should use
    config_to_read = "master" if IS_MASTER else "worker"
    logging.info("Using `%s` configuration" % config_to_read)

    # Opens the stub and reads it
    with open("%s/bgpd_%s.stub" % (FRR_CONFIG_DIR, config_to_read), "r") as bgpd_config_stub_file:
        bgpd_config_stub = bgpd_config_stub_file.read()

    # Write the desired BGP configuration in FRR configuration file
    with open("%s/frr.conf" % FRR_CONFIG_DIR, "w") as frr_config:
        frr_config.write(bgpd_config_stub)

    # Start FRR
    logging.info("Starting FRR daemon.")
    os.system("/etc/init.d/frr start")

    # Remove stubs after FRR is started
    os.remove("%s/bgpd_master.stub" % FRR_CONFIG_DIR)
    os.remove("%s/bgpd_worker.stub" % FRR_CONFIG_DIR)


def start_k8s_watch(v1_client, watch_client):
    global service_ips

    for event in watch_client.stream(
            v1_client.list_namespaced_service,
            namespace='kube-system',
            label_selector='name=kathara-master',
            timeout_seconds=60
    ):
        event_type = event['type']
        kathara_service = event['object']

        ip_address = kathara_service.spec.cluster_ip

        if event_type == "ADDED":
            service_ips.append(ip_address)
            bgp_neighbor("add", ip_address)
        elif event_type == "DELETED":
            service_ips.remove(ip_address)
            bgp_neighbor("del", ip_address)
        elif event_type == "MODIFIED":  # When a node is modified, check if it's ready or not
            for service_ip_address in service_ips:
                bgp_neighbor("del", service_ip_address)

            service_ips = [ip_address]
            bgp_neighbor("add", ip_address)


def bgp_neighbor(event, ip_address):
    logging.info("BGP Neighbor event: %s, Neighbor IP: %s" % (event, ip_address))

    # Adds "no" before the neighbor command if a peer should be removed
    prefix = "" if event == "add" else "no "
    neighbor_string = prefix + (NEIGHBOR_STRING_TEMPLATE % ip_address)

    # Replaces the neighbor command in the vtysh template command
    command_to_execute = [
        x.replace("#neighbor#", neighbor_string) if "#neighbor#" in x else x for x in VTYSH_COMMAND_TEMPLATE
    ]

    # Exec the vtysh command
    os.system(" ".join(command_to_execute))

    # Overwrite current frr.conf
    os.system("vtysh -c 'write'")


if __name__ == '__main__':
    # Init FRR BGP configuration based on current node
    init_frr()

    if not IS_MASTER:
        # Starts Kubernetes Node Watcher on this node
        config.load_incluster_config()

        v1 = client.CoreV1Api()
        w = watch.Watch()

        while True:
            start_k8s_watch(v1, w)
