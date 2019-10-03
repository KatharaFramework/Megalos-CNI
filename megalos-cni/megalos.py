#!/usr/bin/python

import os
import uuid
from pyroute2 import IPRoute, NetNS
from cni_skeleton import skel, result, version

PLUGIN_VERSION = "0.6.5"

BRIDGE_NAME_TEMPLATE = "br-%s"
IPTABLES_COMMAND_TEMPLATE = "iptables -%s FORWARD -o %s -j ACCEPT"
BRIDGE_CONFIG_DIR_TEMPLATE = "/sys/devices/virtual/net/%s/brif/"


def get_default_route_interface_name():
    ip = IPRoute()

    routes = ip.route("dump")

    default_route = [x for x in routes if x["dst_len"] == 0][0]
    default_route_index = [y for (x, y) in default_route["attrs"] if x == "RTA_OIF"][0]

    default_route_interface = ip.link(
        "get",
        index=default_route_index
    )

    if len(default_route_interface) <= 0:
        raise Exception("No default route interface found")

    default_route_interface_name = [y for (x, y) in default_route_interface[0]["attrs"] if x == "IFLA_IFNAME"][0]

    ip.close()

    return default_route_interface_name


def get_master_interface_ip(master_name):
    ip = IPRoute()

    addresses = ip.addr("dump")
    for address in addresses:
        for (x, y) in address["attrs"]:
            if x == "IFA_LABEL" and y == master_name:
                master_ip = [k for (z, k) in address["attrs"] if z == "IFA_ADDRESS"]
                return master_ip[0]

    ip.close()

    raise Exception("Master interface `%s` not found." % master_name)


def iptables_bridge_rule(event, bridge_name):
    cmd = "A" if event == "add" else "D"
    iptables_command = IPTABLES_COMMAND_TEMPLATE % (cmd, bridge_name)

    os.system(iptables_command)


def get_bridge_interfaces_count(bridge_name):
    bridge_directory = BRIDGE_CONFIG_DIR_TEMPLATE % bridge_name
    bridge_interfaces = os.listdir(bridge_directory)

    return len(bridge_interfaces)


def parse_configuration(config):
    name = config["name"] if "name" in config else "vlan" + str(config["vlanId"])    
    master = config["master"] if "master" in config else get_default_route_interface_name()
    suffix = config["suffix"] if "suffix" in config else None

    if "vlanId" not in config:
        raise Exception("`vlanId` parameter should be declared.")

    return name, master, suffix, config["cniVersion"], int(config["vlanId"])


def random_veth_name():
    random = str(uuid.uuid4())
    random = random.replace("-", "")

    return "veth" + random[0:8]


def get_vxlan_name(name, suffix=None):
    vxlan_name = name.replace("net-", "").replace("-", "").replace(".", "")[0:5]

    return vxlan_name + "-" + suffix if suffix is not None else vxlan_name


def parse_mac_address(cni_args):
    try:
        mac_address_list = [x for x in cni_args.split(';') if "MAC=" in x]
        if mac_address_list:
            return mac_address_list.pop().split('=')[1].strip()
    except Exception:
        return None


def create_vxlan_link(name, suffix, master, vlan_id):
    ip = IPRoute()

    vxlan_name = get_vxlan_name(name, suffix)
    vxlan_bridge_name = BRIDGE_NAME_TEMPLATE % vxlan_name

    # Search for desired vxlan bridge
    vxlan_bridge_indexes = ip.link_lookup(ifname=vxlan_bridge_name)

    # If already present, return the bridge interface index
    if len(vxlan_bridge_indexes) > 0:
        return vxlan_bridge_indexes[0]

    # If not, create it. Search the desired master interface index
    interfaces = ip.link_lookup(ifname=master)
    # If not present, raise an error
    if not interfaces:
        raise Exception("Master interface `%s` not found." % master)

    local_ip = get_master_interface_ip(master)

    # Create the vxlan interface on top of the master interface (with vxlan id = VXLAN_INDEX constant)
    ip.link(
        "add",
        ifname=vxlan_name,
        kind="vxlan",
        vxlan_id=vlan_id,
        vxlan_local=local_ip,
        vxlan_learning=False
    )

    # Search the vxlan interface index
    vxlan_index = ip.link_lookup(ifname=vxlan_name)[0]

    # Create the vxlan companion bridge
    ip.link(
        "add",
        ifname=vxlan_bridge_name,
        kind="bridge"
    )

    # Search the vxlan interface index
    vxlan_bridge_index = ip.link_lookup(ifname=vxlan_bridge_name)[0]

    # Attach vxlan interface to the bridge
    ip.link(
        "set",
        index=vxlan_index,
        master=vxlan_bridge_index
    )

    # Bring up the vxlan interface
    ip.link(
        "set",
        index=vxlan_index,
        state="up"
    )

    # Bring up the bridge
    ip.link(
        "set",
        index=vxlan_bridge_index,
        state="up"
    )

    # Adds the filter in IPTables
    iptables_bridge_rule("add", vxlan_bridge_name)

    ip.close()

    # Return the indexes so they can be used by other functions
    return vxlan_bridge_index


# Emulating what veth pair CNI does.
def create_veth_interface(args, vxlan_bridge_index):
    veth_tap_1 = {}

    ip = IPRoute()

    # Due to kernel bug we have to create with temp names or it might collide with the names on the host and error out
    tmp_name_1 = random_veth_name()
    tmp_name_2 = random_veth_name()

    # Create veth pair interface with random veth names
    ip.link(
        "add",
        ifname=tmp_name_1,
        kind="veth",
        peer=tmp_name_2
    )

    # Search for the first veth tap interface
    veth_tap_1_index = ip.link_lookup(ifname=tmp_name_1)[0]
    # Search for the second veth tap interface
    veth_tap_2_index = ip.link_lookup(ifname=tmp_name_2)[0]

    # Attach the second veth tap to the associated vxlan bridge
    ip.link(
        "set",
        index=veth_tap_2_index,
        master=vxlan_bridge_index
    )

    # Change the second veth state to up
    ip.link(
        "set",
        index=veth_tap_2_index,
        state="up"
    )

    # Retrieve the MAC Address from the CNI_ARGS
    mac_address = parse_mac_address(args["CNI_ARGS"])

    # If MAC Address is found in the configuration file, use it.
    if mac_address is not None:
        # Set the MAC Address to the first veth tap (the container one)
        ip.link("set",
                index=veth_tap_1_index,
                address=mac_address
                )

    # Move the first veth tap to the container netNS
    ip.link(
            "set",
            index=veth_tap_1_index,
            net_ns_fd=args["CNI_NETNS"]
    )

    ip.close()

    # Open netNS info
    ns = NetNS(args["CNI_NETNS"])

    # Search for the first veth tap interface in the netNS
    veth_tap_1_index = ns.link_lookup(ifname=tmp_name_1)[0]

    # Change interface name and set it up
    ns.link(
        "set",
        index=veth_tap_1_index,
        ifname=args["CNI_IFNAME"],
        state="up"
    )

    if mac_address is not None:                     # If MAC Address is found in the configuration file, use it.
        veth_tap_1["mac"] = mac_address
    else:                                           # If not, get the generated one.
        # Re-fetch to get all properties/attributes
        cont_macvlan = ns.link("get", index=veth_tap_1_index)[0]
        veth_tap_1["mac"] = [y for (x, y) in cont_macvlan["attrs"] if x == "IFLA_ADDRESS"].pop()

    # Sets values for result
    veth_tap_1["name"] = args["CNI_IFNAME"]
    veth_tap_1["sandbox"] = args["CNI_NETNS"]

    ns.close()

    return veth_tap_1


def delete_vxlan_interface(vxlan_name):
    ip = IPRoute()

    # Search for desired vxlan interface
    vxlan_indexes = ip.link_lookup(ifname=vxlan_name)

    # If already deleted, skip
    if len(vxlan_indexes) > 0:
        vxlan_index = vxlan_indexes[0]

        # Destroy the vxlan interface
        ip.link(
            "del",
            index=vxlan_index
        )

    # Search the vxlan bridge index
    vxlan_bridge_name = BRIDGE_NAME_TEMPLATE % vxlan_name
    vxlan_bridge_indexes = ip.link_lookup(ifname=vxlan_bridge_name)

    # If already deleted, skip
    if len(vxlan_bridge_indexes) > 0:
        vxlan_bridge_index = vxlan_bridge_indexes[0]

        # Destroy the vxlan bridge
        ip.link(
            "del",
            index=vxlan_bridge_index
        )

        # Delete the filter in IPTables
        iptables_bridge_rule("del", vxlan_bridge_name)


def delete_veth_interface(args):
    ns = NetNS(args["CNI_NETNS"])

    # Search for the desired interface
    macvlan_indexes = ns.link_lookup(ifname=args["CNI_IFNAME"])

    # Don't do nothing if interface is not present
    if len(macvlan_indexes) <= 0:
        return

    macvlan_index = macvlan_indexes[0]

    # Destroy the interface
    ns.link(
        "del",
        index=macvlan_index
    )

    ns.close()


def cmd_add(args):
    try:
        name, master, suffix, cni_version, vlan_id = parse_configuration(args["stdin"])

        vxlan_bridge_index = create_vxlan_link(name, suffix, master, vlan_id)
        veth_interface = create_veth_interface(args, vxlan_bridge_index)

        return result.print_result(
            result.create_result(
                cni_version,
                interfaces=[veth_interface]
            )
        )
    except Exception as e:
        return result.create_error(137, e.message)


def cmd_check(args):
    # TODO: https://github.com/containernetworking/plugins/blob/master/plugins/main/macvlan/macvlan.go#L332
    pass


def cmd_del(args):
    try:
        name, _, suffix, _, _ = parse_configuration(args["stdin"])

        # Delete the container's veth interface
        delete_veth_interface(args)

        # Check if the vxlan interface (and the companion bridge) should be deleted
        # Generate associated vxlan names
        vxlan_name = get_vxlan_name(name, suffix)
        vxlan_bridge_name = BRIDGE_NAME_TEMPLATE % vxlan_name
        # Get the remaining interfaces attached to the bridge
        new_counter = get_bridge_interfaces_count(vxlan_bridge_name)

        # If the counter is 1 (means that only the vxlan interface is attached), delete the vxlan interface
        # and the companion bridge
        if new_counter <= 1:
            delete_vxlan_interface(vxlan_name)
    except Exception as e:
        return result.create_error(138, e.message)


if __name__ == '__main__':
    skel.plugin_main(cmd_add, cmd_check, cmd_del, version.all_versions, version.build_string("megalos", PLUGIN_VERSION))
