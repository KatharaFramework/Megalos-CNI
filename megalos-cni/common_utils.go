package main

import (
	"io/ioutil"
	"strings"
)

func getMacAddress(args string) string {
	splitArgs := strings.Split(args, ";")
	for _, arg := range splitArgs {
		if strings.Contains(arg, "MAC=") {
			return strings.TrimSpace(strings.Split(arg, "=")[1])
		}
	}

	return ""
}

func getBridgeInterfacesCount(bridgeName string) (int, error) {
	bridgeDirectory := "/sys/devices/virtual/net/" + bridgeName + "/brif/"
	bridgeInterfaces, err := ioutil.ReadDir(bridgeDirectory)
	if err != nil {
		return -1, err
	}

	return len(bridgeInterfaces), nil
}