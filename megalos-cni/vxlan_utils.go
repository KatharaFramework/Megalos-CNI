package main

import (
	"fmt"
	"net"
	"os"
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

func createVxlanLink(name string, suffix string, master string, vxlanId int) (netlink.Link, error) {
	vxlanName, vxlanBridgeName := getVxlanAndBridgeName(name, suffix)

	bridgeCreated := true

	// Search the desired master interface index
	var masterLink netlink.Link
	var err error
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

	vxlan, err := createVxlanInterface(vxlanName, vxlanId, masterInterfaceIP)
	if err != nil {
		return nil, err
	}

	// Create the vxlan companion bridge
	vxlanBridgeLinkAttrs := netlink.NewLinkAttrs()
	vxlanBridgeLinkAttrs.Name = vxlanBridgeName
	err = netlink.LinkAdd(&netlink.Bridge{
		LinkAttrs: 	vxlanBridgeLinkAttrs,
	})
	switch {
		// No errors
		case err == nil:
			break

		// Bridge already exists, flag that it is not created
		case os.IsExist(err):
			bridgeCreated = false

		// Raise other errors
		default:
			return nil, fmt.Errorf("failed to create VXLAN companion bridge %q: %v", vxlanBridgeName, err)
	}

	// Get the vxlan bridge
	vxlanBridge, err := netlink.LinkByName(vxlanBridgeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get VXLAN companion bridge %q: %v", vxlanBridgeName, err)
	}

	// If the interface or bridge are new, attach the vxlan interface to the bridge
	if vxlan != nil || bridgeCreated {
		// Attach the vxlan interface to the bridge
		if err = attachInterfaceToBridge(vxlanBridge, vxlan); err != nil {
			return nil, err
		}
	}

	// If the bridge is new, add the iptables rule
	if bridgeCreated {
		// Add a bridge forward rule to iptables
		outRule := iptRule{table: iptables.Filter,
			chain: "FORWARD",
			args: []string{"-o", vxlanBridgeName, "-j", "ACCEPT"},
		}
		if err = programChainRule(outRule, true); err != nil {
			return nil, fmt.Errorf("failed to add iptables rule of %q: %v", vxlanBridgeName, err)
		}
	}

	return vxlanBridge, nil
}

func deleteVxlanLink(vxlanName string, vxlanBridgeName string) error {
	// Search for desired vxlan interface
	vxlanLink, _ := netlink.LinkByName(vxlanName)

	// If already deleted, skip
	if vxlanLink != nil {
		if err := netlink.LinkDel(vxlanLink); err != nil {
			return fmt.Errorf("failed to delete VXLAN interface %q: %v", vxlanName, err)
		}
	}

	// Search the vxlan bridge index
	vxlanBridge, _ := netlink.LinkByName(vxlanBridgeName)
	if vxlanBridge != nil {
		if err := netlink.LinkDel(vxlanBridge); err != nil {
			return fmt.Errorf("failed to delete VXLAN companion bridge %q: %v", vxlanBridgeName, err)
		}
	}

	outRule := iptRule{table: iptables.Filter,
					   chain: "FORWARD",
					   args: []string{"-o", vxlanBridgeName, "-j", "ACCEPT"},
					   }
	if err := programChainRule(outRule, false); err != nil {
		return fmt.Errorf("failed to delete iptables rule of %q: %v", vxlanBridgeName, err)
	}

	return nil
}

func createVxlanInterface(vxlanName string, vxlanId int, masterInterfaceIP net.IP) (netlink.Link, error) {
	// Create the vxlan interface on top of the master interface
	vxlanLinkAttrs := netlink.NewLinkAttrs()
	vxlanLinkAttrs.Name = vxlanName
	err := netlink.LinkAdd(&netlink.Vxlan{
		LinkAttrs: 	vxlanLinkAttrs,
		VxlanId: 	vxlanId,
		SrcAddr: 	masterInterfaceIP,
		Learning: 	false,
	})

	switch {
		// No errors
		case err == nil:
			// Get the vxlan interface
			vxlan, err := netlink.LinkByName(vxlanName)
			if err != nil {
				return nil, fmt.Errorf("failed to get VXLAN interface %q: %v", vxlanName, err)
			}

			return vxlan, nil

		// Interface already exists
		case os.IsExist(err):
			vxlan, err := netlink.LinkByName(vxlanName)
			if err != nil {
				return nil, fmt.Errorf("VXLAN interface with the same VNI (%d) already exists", vxlanId)
			}

			// If interface exists and the vxlan id is the same requested, return nil so interface is not created
			if vxlanId == vxlan.(*netlink.Vxlan).VxlanId {
				return nil, nil
			} else {
				// If interface exists but the vxlan id is not the same requested, raise error
				return nil, fmt.Errorf("VXLAN interface %q has a different VXLAN tag: %v", vxlanName, err)
			}

		// Raise other errors
		default:
			return nil, fmt.Errorf("failed to create VXLAN interface %q: %v", vxlanName, err)
	}
}