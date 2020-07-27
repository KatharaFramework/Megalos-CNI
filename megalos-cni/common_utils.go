package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"io/ioutil"
)

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

func attachInterfaceToBridge(bridge netlink.Link, iface netlink.Link) error {
	interfaceName := iface.Attrs().Name
	bridgeName := bridge.Attrs().Name

	if err := netlink.LinkSetMaster(iface, bridge); err != nil {
		return fmt.Errorf("failed to set master of %q to %q: %v", interfaceName, bridgeName, err)
	}

	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("failed to set interface %q up: %v", interfaceName, err)
	}

	if err := netlink.LinkSetUp(bridge); err != nil {
		return fmt.Errorf("failed to set bridge %q up: %v", bridgeName, err)
	}

	return nil
}

func getBridgeInterfacesCount(bridgeName string) (int, error) {
	bridgeDirectory := "/sys/devices/virtual/net/" + bridgeName + "/brif/"
	bridgeInterfaces, err := ioutil.ReadDir(bridgeDirectory)
	if err != nil {
		return -1, err
	}

	return len(bridgeInterfaces), nil
}