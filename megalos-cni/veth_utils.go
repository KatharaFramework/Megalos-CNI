package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
)

const (
	vethPrefix	= "veth"
	vethLen		= 8
)

func randomVethName() string {
	uuidString, _ := uuid.NewRandom()

	return vethPrefix + strings.Replace(uuidString.String(), "-", "", -1)[:vethLen]
}

func createVethPair(args *skel.CmdArgs, conf *MegalosConf, vxlanBridgeInterface netlink.Link) (*current.Interface, *current.Interface, error) {
	veth1Name := randomVethName()
	veth2Name := randomVethName()

	containerInterface := &current.Interface{}
	hostInterface := &current.Interface{}

	containerInterface.Name = args.IfName
	hostInterface.Name = veth2Name

	vethMacAddressString := getMacAddress(args.Args)

	// Retrieve the MAC Address from the CNI args
	var vethMacAddress net.HardwareAddr
	var err error
	if vethMacAddressString != "" {
		vethMacAddress, err = net.ParseMAC(vethMacAddressString)
		if err != nil {
			return nil, nil, err
		}
	} else {
		vethMacAddress = nil
	}

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = veth1Name
	// If MAC Address is found in the CNI args, use it.
	if vethMacAddress != nil {
		linkAttrs.HardwareAddr = vethMacAddress
	}

	if err = netlink.LinkAdd(&netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  veth2Name,
	}); err != nil {
		return nil, nil, err
	}

	// Get first veth tap interface
	veth1Link, err := netlink.LinkByName(veth1Name)
	if err != nil {
		return nil, nil, err
	}

	// Get second veth tap interface
	veth2Link, err := netlink.LinkByName(veth2Name)
	if err != nil {
		return nil, nil, err
	}

	// Attach the second veth tap to the associated vxlan bridge
	if err = attachInterfaceToBridge(vxlanBridgeInterface, veth2Link); err != nil {
		return nil, nil, err
	}

	containerInterface.Mac = veth1Link.Attrs().HardwareAddr.String()
	hostInterface.Mac = veth2Link.Attrs().HardwareAddr.String()

	// Get the container netNS
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()
	containerInterface.Sandbox = netns.Path()

	// Move the first veth tap to the container netNS
	if err = netlink.LinkSetNsFd(veth1Link, int(netns.Fd())); err != nil {
		return nil, nil, fmt.Errorf("failed to move %q to host netns: %v", veth1Name, err)
	}

	err = netns.Do(func(hostNS ns.NetNS) error {
		// Search for the first veth interface in the netNS
		veth1Link, err = netlink.LinkByName(veth1Name)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", veth1Name, hostNS.Path(), err)
		}

		// Rename interface name and set it up
		if err = netlink.LinkSetName(veth1Link, args.IfName); err != nil {
			return fmt.Errorf("failed to rename %q in %q in %q: %v", veth1Name, args.IfName, hostNS.Path(), err)
		}

		if err = netlink.LinkSetUp(veth1Link); err != nil {
			return fmt.Errorf("failed to set %q up: %v", veth1Name, err)
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return hostInterface, containerInterface, nil
}

func deleteVethPair(args *skel.CmdArgs) error {
	// Get the container netNS
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	err = netns.Do(func(hostNS ns.NetNS) error {
		// Search for the interface in the netNS
		vethLink, err := netlink.LinkByName(args.IfName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", args.IfName, hostNS.Path(), err)
		}

		// Delete the interface
		if err = netlink.LinkDel(vethLink); err != nil {
			return fmt.Errorf("failed to set %q up: %v", args.IfName, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
