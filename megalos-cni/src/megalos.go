package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/040"
	"github.com/containernetworking/cni/pkg/version"

	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

type MegalosConf struct {
	// Basic CNI Configuration
	types.NetConf

	// Specific CNI Configuration
	Master 	   		string 		`json:"master,omitempty"`
	Suffix      	string   	`json:"suffix"`
	VxlanId    		int 		`json:"vxlanId"`
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func parseConfig(stdin []byte) (*MegalosConf, error) {
	conf := MegalosConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	return &conf, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	vxlanBridge, err := createVxlanLink(conf.Name, conf.Suffix, conf.Master, conf.VxlanId)
	if err != nil {
		return err
	}

	hostInterface, containerInterface, err := createVethPair(args, conf, vxlanBridge)
	if err != nil {
		return err
	}

	result := &types040.Result{
		CNIVersion: conf.CNIVersion,
		Interfaces: []*types040.Interface{hostInterface, containerInterface},
	}

	return types.PrintResult(result, conf.CNIVersion)
}

// cmdDel is called for DELETE requests
func cmdDel(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}

	// Delete the container's veth interface
	if err := deleteVethPair(args); err != nil {
		return err
	}

	// Check if the vxlan interface (and the companion bridge) should be deleted
	// Generate associated vxlan names
	vxlanName, vxlanBridgeName := getVxlanAndBridgeName(conf.Name, conf.Suffix)
	// Get the remaining interfaces attached to the bridge
	newCounter, err := getBridgeInterfacesCount(vxlanBridgeName)
	if err != nil {
		return err
	}


	// If the counter is 1 (means that only the vxlan interface is attached), delete the vxlan interface
	// and the companion bridge
	if newCounter <= 1 {
		if err := deleteVxlanLink(vxlanName, vxlanBridgeName); err != nil {
			return err
		}
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return fmt.Errorf("not implemented")
}


func main() {
	bv.BuildVersion = "0.8.5"
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("megalos"))
}
