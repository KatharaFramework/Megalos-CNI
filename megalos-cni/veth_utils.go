package main

import (
	"fmt"
	"os"
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
	veth1Name, veth2Name, err := makeVeth()
	if err != nil {
		return nil, nil, err
	}

	containerInterface := &current.Interface{}
	hostInterface := &current.Interface{}

	containerInterface.Name = args.IfName
	hostInterface.Name = veth2Name

	// Get first veth interface
	veth1Link, err := netlink.LinkByName(veth1Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get veth1 %q: %v", veth1Name, err)
	}

	// Get second veth interface
	veth2Link, err := netlink.LinkByName(veth2Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get veth2 %q: %v", veth2Name, err)
	}

	containerInterface.Mac = veth1Link.Attrs().HardwareAddr.String()
	hostInterface.Mac = veth2Link.Attrs().HardwareAddr.String()

	// Attach the second veth to the associated vxlan bridge
	if err = attachInterfaceToBridge(vxlanBridgeInterface, veth2Link); err != nil {
		return nil, nil, err
	}

	// Get the container netNS
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()
	containerInterface.Sandbox = netns.Path()

	// Move the first veth to the container netNS
	if err = netlink.LinkSetNsFd(veth1Link, int(netns.Fd())); err != nil {
		return nil, nil, fmt.Errorf("failed to move %q to netns: %v", veth1Name, err)
	}

	// Access the netNS
	err = netns.Do(func(hostNS ns.NetNS) error {
		// Search for the first veth interface in the netNS
		veth1NsLink, err := netlink.LinkByName(veth1Name)
		if err != nil {
			return fmt.Errorf("failed to lookup veth1 %q in %q: %v", veth1Name, hostNS.Path(), err)
		}

		// Rename interface veth1
		if err = netlink.LinkSetName(veth1NsLink, args.IfName); err != nil {
		 	return fmt.Errorf("failed to rename veth1 %q in %q in %q: %v", veth1Name, args.IfName, hostNS.Path(), err)
		}

		// Search for the renamed veth1
		veth1NsLink, err = netlink.LinkByName(args.IfName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", args.IfName, hostNS.Path(), err)
		}

		if err = netlink.LinkSetUp(veth1NsLink); err != nil {
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
			return fmt.Errorf("failed to delete %q: %v", args.IfName, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func makeVeth() (string, string, error) {
	for i := 0; i < 10; i++ {
		veth1Name := randomVethName()
		veth2Name := randomVethName()

		linkAttrs := netlink.NewLinkAttrs()
		linkAttrs.Name = veth1Name

		err := netlink.LinkAdd(&netlink.Veth{
			LinkAttrs: linkAttrs,
			PeerName:  veth2Name,
		})

		switch {
			case err == nil:
				return veth1Name, veth2Name, nil

			case os.IsExist(err):
				if peerExists(veth2Name) {
					continue
				}

			default:
				return "", "", fmt.Errorf("failed to make veth pair: %v", err)
		}
	}

	// Should really never be hit
	return "", "", fmt.Errorf("failed to find a unique veth name")
}

func peerExists(name string) bool {
	if _, err := netlink.LinkByName(name); err != nil {
		return false
	}

	return true
}