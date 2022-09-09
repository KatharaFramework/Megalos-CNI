#!/usr/bin/make -f

.PHONY: gobuild

gobuild_%:
	go mod download
	GOOS=linux CGO_ENABLED=0 GOARCH=$* go build src/megalos.go src/common_utils.go src/vxlan_utils.go src/veth_utils.go
	mv ./megalos ./megalos-$*
	chmod 777 ./megalos-$*