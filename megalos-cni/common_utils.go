package main

import (
	"io/ioutil"
)

func getBridgeInterfacesCount(bridgeName string) (int, error) {
	bridgeDirectory := "/sys/devices/virtual/net/" + bridgeName + "/brif/"
	bridgeInterfaces, err := ioutil.ReadDir(bridgeDirectory)
	if err != nil {
		return -1, err
	}

	return len(bridgeInterfaces), nil
}