package main

import (
	"fmt"
	"strings"

	"github.com/docker/libnetwork/iptables"
	"github.com/vishvananda/netlink"
)

const (
	vxlanLen 		= 5
	bridgePrefix    = "kt"
)

func getVxlanAndBridgeName(name string, suffix string) (string, string) {
	vxlanName := strings.Replace(name, "_", "", -1)
	// Cut name if greater than five chars
	if len(vxlanName) > 5 {
		vxlanName = vxlanName[:vxlanLen]
	}

	if suffix != "" {
		vxlanName = vxlanName + "-" + suffix
	}

	return vxlanName, bridgePrefix + "-" + vxlanName
}

func getDefaultRouteInterfaceName() (int, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return -1, err
	}

	for _, route := range routes {
		if route.Dst == nil {
			return route.LinkIndex, nil
		}
	}

	return -1, fmt.Errorf("can not find default route interface")
}

func createVxlanLink(name string, suffix string, master string, vxlanId int) (netlink.Link, error) {
	vxlanName, vxlanBridgeName := getVxlanAndBridgeName(name, suffix)

	// Search for desired vxlan bridge
	vxlanBridge, err := netlink.LinkByName(vxlanBridgeName)
	if vxlanBridge != nil {
		return vxlanBridge, nil
	}

	// Search the desired master interface index
	var masterLink netlink.Link
	if master == "" {
		masterIndex, err := getDefaultRouteInterfaceName()
		if err != nil {
			return nil, err
		}

		masterLink, err = netlink.LinkByIndex(masterIndex)
	} else {
		masterLink, err = netlink.LinkByName(master)
		if err != nil {
			return nil, err
		}
	}

	// Get master interface IP
	addresses, err := netlink.AddrList(masterLink, netlink.FAMILY_V4)
	masterInterfaceIP := addresses[0].IP

	// Create the vxlan interface on top of the master interface
	vxlanLinkAttrs := netlink.NewLinkAttrs()
	vxlanLinkAttrs.Name = vxlanName
	if err = netlink.LinkAdd(&netlink.Vxlan{
		LinkAttrs: 	vxlanLinkAttrs,
		VxlanId: 	vxlanId,
		SrcAddr: 	masterInterfaceIP,
		Learning: 	false,
	}); err != nil {
		return nil, err
	}

	// Get the vxlan interface
	vxlan, err := netlink.LinkByName(vxlanName)
	if err != nil {
		return nil, err
	}

	// Create the vxlan companion bridge
	vxlanBridgeLinkAttrs := netlink.NewLinkAttrs()
	vxlanBridgeLinkAttrs.Name = vxlanBridgeName
	if err = netlink.LinkAdd(&netlink.Bridge{
		LinkAttrs: 	vxlanBridgeLinkAttrs,
	}); err != nil {
		return nil, err
	}

	// Get the vxlan bridge
	vxlanBridge, err = netlink.LinkByName(vxlanBridgeName)
	if err != nil {
		return nil, err
	}

	if err = attachInterfaceToBridge(vxlanBridge, vxlan); err != nil {
		return nil, err
	}

	outRule := iptRule{table: iptables.Filter,
			           chain: "FORWARD",
					   args: []string{"-o", vxlanBridgeName, "-j", "ACCEPT"},
					   }
	if err = programChainRule(outRule, true); err != nil {
		return nil, err
	}

	return vxlanBridge, nil
}

func deleteVxlanLink(vxlanName string, vxlanBridgeName string) error {
	// Search for desired vxlan interface
	vxlanLink, _ := netlink.LinkByName(vxlanName)

	// If already deleted, skip
	if vxlanLink != nil {
		if err := netlink.LinkDel(vxlanLink); err != nil {
			return err
		}
	}

	// Search the vxlan bridge index
	vxlanBridge, _ := netlink.LinkByName(vxlanBridgeName)
	if vxlanBridge != nil {
		if err := netlink.LinkDel(vxlanBridge); err != nil {
			return err
		}
	}

	outRule := iptRule{table: iptables.Filter,
					   chain: "FORWARD",
					   args: []string{"-o", vxlanBridgeName, "-j", "ACCEPT"},
					   }
	if err := programChainRule(outRule, false); err != nil {
		return err
	}

	return nil
}

func attachInterfaceToBridge(bridge netlink.Link, iface netlink.Link) error {
	if err := netlink.LinkSetMaster(iface, bridge); err != nil {
		return err
	}

	if err := netlink.LinkSetUp(iface); err != nil {
		return err
	}

	if err := netlink.LinkSetUp(bridge); err != nil {
		return err
	}

	return nil
}